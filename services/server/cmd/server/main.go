// Command server 九转修仙 - 统一单体服务入口。
//
// 将原微服务架构(网关 + 9个领域微服务)合并为一个单体进程。
// 通过 adapter 包桥接各服务模块 internal 包，共享 MySQL/Redis/MongoDB。
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"net/http"
	"os"
	"strings"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	// ---- Gateway 包 ----
	gatewayadapter "cultivation-game/services/gateway/pkg/adapter"

	// ---- Service Adapters ----
	authadapter "cultivation-game/services/auth/pkg/adapter"
	combatadapter "cultivation-game/services/combat/pkg/adapter"
	cultivationadapter "cultivation-game/services/cultivation/pkg/adapter"
	playeradapter "cultivation-game/services/player/pkg/adapter"
	rankingadapter "cultivation-game/services/ranking/pkg/adapter"
	socialadapter "cultivation-game/services/social/pkg/adapter"
	tradeadapter "cultivation-game/services/trade/pkg/adapter"
	worldadapter "cultivation-game/services/world/pkg/adapter"


	// ---- 第三方包 ----
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ============================================================================
// 服务上下文 — 后续应重构为 struct 以支持依赖注入和单元测试(TODO)
// ============================================================================
var startedAt = time.Now()
var serverRegistered atomic.Int64 // 总注册人数（缓存）
var appDB *sql.DB                 // 全局 DB 引用，供 wsHandler 等使用

func main() {
	// ---------------------------------------------------------------
	// 0. 加载配置
	// ---------------------------------------------------------------
	mysqlDSN := getEnv("MYSQL_DSN", "root:password@tcp(127.0.0.1:3306)/cultivation?charset=utf8mb4&parseTime=True&loc=Local")
	redisAddr := getEnv("REDIS_ADDR", "127.0.0.1:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := getEnvInt("REDIS_DB", 0)
	mongoURI := os.Getenv("MONGO_URI")
	serverPort := getEnv("SERVER_PORT", "8080")
	jwtAccessSecret := getEnv("JWT_ACCESS_SECRET", "default-access-secret-change-in-production")
	jwtRefreshSecret := getEnv("JWT_REFRESH_SECRET", "default-refresh-secret-change-in-production")

	// 安全校验：拒绝使用默认值的 JWT 密钥
	if jwtAccessSecret == "default-access-secret-change-in-production" || jwtRefreshSecret == "default-refresh-secret-change-in-production" {
		slog.Error("JWT 密钥使用了不安全的默认值，请在环境变量中设置 JWT_ACCESS_SECRET 和 JWT_REFRESH_SECRET")
		os.Exit(1)
	}

	jwtAccessExpire := getDuration("JWT_ACCESS_EXPIRE", 24*time.Hour)
	jwtRefreshExpire := getDuration("JWT_REFRESH_EXPIRE", 7*24*time.Hour)
	jwtIssuer := getEnv("JWT_ISSUER", "cultivation-game")

	// ---------------------------------------------------------------
	// 1. 初始化日志
	// ---------------------------------------------------------------
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
	slog.Info("unified server starting", "port", serverPort)

	// ---------------------------------------------------------------
	// 2. 连接 MySQL
	// ---------------------------------------------------------------
	var err error; appDB, err = sql.Open("mysql", mysqlDSN)
	if err != nil {
		slog.Error("failed to open MySQL", "error", err)
		os.Exit(1)
	}
	defer appDB.Close()
	appDB.SetMaxOpenConns(100)
	appDB.SetMaxIdleConns(20)
	appDB.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := appDB.PingContext(ctx); err != nil {
		cancel()
		slog.Warn("MySQL ping failed, running in degraded mode", "error", err)
	} else {
		slog.Info("MySQL connected")
	}
	cancel()

	// ---------------------------------------------------------------
	// 3. 连接 Redis
	// ---------------------------------------------------------------
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})
	defer rdb.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 3*time.Second)
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		pingCancel()
		slog.Warn("Redis ping failed, running in degraded mode", "error", err)
	} else {
		slog.Info("Redis connected")
	}
	pingCancel()

	// ---------------------------------------------------------------
	// 4. 连接 MongoDB (可选)
	// ---------------------------------------------------------------
	var mongoClient *mongo.Client
	var mongoDB *mongo.Database
	hasMongo := false
	if mongoURI != "" {
		mongoCtx, mongoCancel := context.WithTimeout(context.Background(), 10*time.Second)
		mongoClient, mongoErr := mongo.Connect(mongoCtx, options.Client().ApplyURI(mongoURI))
		mongoCancel()
		if mongoErr != nil {
			slog.Warn("MongoDB connection failed", "error", mongoErr)
		} else {
			mongoDB = mongoClient.Database("cultivation_game")
			hasMongo = true
			slog.Info("MongoDB connected")
		}
	} else {
		slog.Warn("MONGO_URI not set, social routes will be skipped")
	}

	// ---------------------------------------------------------------
	// 5. 初始化 Auth 服务 (via adapter)
	// ---------------------------------------------------------------
	slog.Info("initializing auth service")
	auth := authadapter.Bootstrap(appDB, rdb, slog.Default())
	_ = auth // AuthService and GMHandler used below

	// Seed default GM admin
	if err := auth.SeedDefaultAdmin(); err != nil {
		slog.Error("failed to seed default GM admin", "error", err)
	} else {
		slog.Info("GM admin ready")
	}

	if err := initAuthSchema(appDB); err != nil {
		slog.Error("failed to init auth schema", "error", err)
		os.Exit(1)
	}

	// ---------------------------------------------------------------
	// 6. 初始化 JWT 管理器
	// ---------------------------------------------------------------
	jwtManager := gatewayadapter.NewJWTManager(
		jwtAccessSecret, jwtRefreshSecret,
		jwtAccessExpire, jwtRefreshExpire, jwtIssuer,
	)

	// ---------------------------------------------------------------
	// 7. 初始化 Player 服务 (via adapter)
	// ---------------------------------------------------------------
	slog.Info("initializing player service")
	playerDataDir := filepath.Join("..", "player", "internal", "data")
	player := playeradapter.Bootstrap(appDB, rdb, playerDataDir)

	// ---------------------------------------------------------------
	// 8. 初始化 Combat 服务 (via adapter)
	// ---------------------------------------------------------------
	slog.Info("initializing combat service")
	combatDataDir := filepath.Join("..", "combat", "internal", "data")
	playerAddr := fmt.Sprintf("http://127.0.0.1:%s", serverPort)
	combat := combatadapter.Bootstrap(combatDataDir, playerAddr)
	defer combat.Stop()

	// ---------------------------------------------------------------
	// 9. 初始化 Cultivation 服务 (via adapter)
	// ---------------------------------------------------------------
	slog.Info("initializing cultivation service")
	cultDataDir := getEnv("CULTIVATION_DATA_DIR", filepath.Join("..", "cultivation", "internal", "data"))
	cultivation := cultivationadapter.Bootstrap(appDB, rdb, cultDataDir, playerAddr)
	defer cultivation.TickerSvc.Stop()

	// ---------------------------------------------------------------
	// 10. 初始化 Social 服务 (via adapter)
	// ---------------------------------------------------------------
	slog.Info("initializing social service")
	social := socialadapter.Bootstrap(mongoDB, rdb, playerAddr)
	if social != nil {
		slog.Info("social service initialized")
	} else {
		slog.Warn("social service skipped (MongoDB required)")
	}

	// ---------------------------------------------------------------
	// 11. 初始化 World 服务 (via adapter)
	// ---------------------------------------------------------------
	slog.Info("initializing world service")
	worldDataDir := filepath.Join("..", "world", "internal", "data")
	world := worldadapter.Bootstrap(rdb, worldDataDir)
	defer world.BossSvc.Stop()

	// ---------------------------------------------------------------
	// 12. 初始化 Trade 服务 (via adapter)
	// ---------------------------------------------------------------
	slog.Info("initializing trade service")
	trade := tradeadapter.Bootstrap(appDB, rdb)
	defer trade.AuctionCancel()


	// ---------------------------------------------------------------
	// 13. 初始化 Ranking 服务 (via adapter)
	// ---------------------------------------------------------------
	slog.Info("initializing ranking service")
	ranking := rankingadapter.Bootstrap(rdb, playerAddr)
	defer ranking.Service.Stop()

	// ---------------------------------------------------------------
	// 14. 创建 Gin 路由并注册所有路由
	// ---------------------------------------------------------------
	slog.Info("registering routes")
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogger())
	r.Use(corsMiddleware())

	// ---- 健康检查 ----
	r.GET("/health", func(c *gin.Context) {

		dbStatus := "ok"
		if err := appDB.Ping(); err != nil {
			dbStatus = "degraded"
		}
		redisStatus := "ok"
		if err := rdb.Ping(c.Request.Context()).Err(); err != nil {
			redisStatus = "degraded"
		}
		if serverRegistered.Load() == 0 {
			var cnt int64
			if err := appDB.QueryRow("SELECT COUNT(*) FROM players").Scan(&cnt); err != nil {
				serverRegistered.Store(0)
			} else {
				serverRegistered.Store(cnt)
			}
		}
		var online int64
		appDB.QueryRow("SELECT COUNT(*) FROM players WHERE updated_at > DATE_SUB(NOW(), INTERVAL 5 MINUTE)").Scan(&online)
		status := gin.H{
			"status":     "ok",
			"service":    "unified-server",
			"mysql":      dbStatus,
			"redis":      redisStatus,
			"online":     online,
			"registered": serverRegistered.Load(),
		}
		if hasMongo {
			mongoStatus := "ok"
			if mongoClient != nil {
				if err := mongoClient.Ping(c.Request.Context(), nil); err != nil {
					mongoStatus = "degraded"
				}
			} else {
				mongoStatus = "disconnected"
			}
			status["mongodb"] = mongoStatus
		}
		c.JSON(http.StatusOK, status)
	})

	// ---- WebSocket 端点 ----
	r.GET("/ws", wsHandler(jwtManager))

	// ---- Auth 端点 ----
	r.POST("/auth/login", loginHandler(jwtManager, auth))
	r.POST("/auth/register", registerHandler(jwtManager, auth))
	r.POST("/auth/refresh", refreshTokenHandler(jwtManager))

	// ---- GM 管理后台 ----
	gmV1 := r.Group("/api/v1/gm")
	{
		gmV1.POST("/login", auth.GMHandler.Login)

		authed := gmV1.Group("")
		authed.Use(auth.GMAuthMiddleware())
		{
			authed.GET("/players", auth.GMHandler.GetPlayerList)
			authed.GET("/players/:id", auth.GMHandler.GetPlayerDetail)

			write := authed.Group("")
			write.Use(auth.GMPermissionMiddleware())
			{
				write.PUT("/players/:id/attribute", auth.GMHandler.EditPlayerAttribute)
				write.POST("/players/:id/ban", auth.GMHandler.BanPlayer)
				write.DELETE("/players/:id/ban", auth.GMHandler.UnbanPlayer)
				write.POST("/announcements", auth.GMHandler.SendAnnouncement)
				write.POST("/players/:id/items", auth.GMHandler.SendItem)
			}

			authed.GET("/stats", auth.GMHandler.GetServerStats)
			authed.GET("/logs", auth.GMHandler.GetOperationLogs)
		}
	}

	// ---- Player 服务路由 ----
	p := player
	v1 := r.Group("/api/v1/player")
	{
		v1.POST("/register", p.Player.Register)
		v1.GET("/:id", p.Player.GetProfile)
		v1.PUT("/:id", p.Player.UpdateProfile)
		v1.GET("/user/:user_id", p.Player.GetByUserID)
		v1.POST("/:id/currency", p.Player.UpdateCurrency)
		v1.POST("/:id/update-realm", p.Player.UpdateRealm)
		v1.POST("/:id/add-exp", p.Player.AddExp)
		v1.POST("/:id/update-exp", p.Player.UpdateExp)
		v1.POST("/:id/update-attributes", p.Player.UpdateAttributes)
			v1.POST("/:id/meditation", p.Player.MeditationAction)
		v1.POST("/:id/add-item", p.Player.AddItem)
		v1.POST("/:id/remove-item", p.Player.RemoveItem)
			v1.POST("/:id/cultivate/tick", jwtAuthMiddleware(jwtManager), p.Cultivation.CultivateTick)
			v1.POST("/:id/breakthrough", jwtAuthMiddleware(jwtManager), p.Cultivation.Breakthrough)
		v1.POST("/:id/train", jwtAuthMiddleware(jwtManager), p.Training.Train)
		v1.POST("/:id/pve", jwtAuthMiddleware(jwtManager), p.Pve.Fight)
		v1.GET("/:id/inventory", jwtAuthMiddleware(jwtManager), p.Backpack.ListInventory)
		v1.GET("/:id/equipment", jwtAuthMiddleware(jwtManager), p.Equip.ListEquipment)
		v1.POST("/:id/equipment/craft", jwtAuthMiddleware(jwtManager), p.Equip.CraftEquipment)
		v1.POST("/:id/equipment/unequip", jwtAuthMiddleware(jwtManager), p.Equip.Unequip)
		v1.GET("/:id/equipment/templates", jwtAuthMiddleware(jwtManager), p.Equip.GetTemplates)
		v1.GET("/:id/pills", jwtAuthMiddleware(jwtManager), p.Pill.ListPlayerPills)
		v1.POST("/:id/pills/craft", jwtAuthMiddleware(jwtManager), p.Pill.CraftPill)
		v1.POST("/:id/pills/use", jwtAuthMiddleware(jwtManager), p.Pill.UsePill)
		v1.GET("/pills/recipes", jwtAuthMiddleware(jwtManager), p.Pill.ListRecipes)
		v1.GET("/:id/friends", jwtAuthMiddleware(jwtManager), p.Friend.ListFriends)
		v1.POST("/:id/friends/add", jwtAuthMiddleware(jwtManager), p.Friend.AddFriend)
		v1.POST("/:id/friends/accept", jwtAuthMiddleware(jwtManager), p.Friend.AcceptFriend)
		v1.POST("/:id/friends/remove", jwtAuthMiddleware(jwtManager), p.Friend.RemoveFriend)
		v1.GET("/:id/friends/pending", jwtAuthMiddleware(jwtManager), p.Friend.PendingRequests)
		v1.GET("/:id/friends/search", jwtAuthMiddleware(jwtManager), p.Friend.SearchPlayers)
		v1.GET("/:id/messages", jwtAuthMiddleware(jwtManager), p.Friend.GetPrivateMessages)
		v1.POST("/:id/messages/send", jwtAuthMiddleware(jwtManager), p.Friend.SendPrivateMessage)
		v1.GET("/:id/attributes", p.Player.GetAttributes)



		dongfu := v1.Group("/:id/dongfu")
		{
			dongfu.POST("/build", p.DongFu.Build)
			dongfu.GET("", p.DongFu.GetDongFu)
			dongfu.POST("/room/build", p.DongFu.BuildRoom)
			dongfu.POST("/room/upgrade", p.DongFu.UpgradeRoom)
			dongfu.GET("/room/:room_id", p.DongFu.GetRoomDetail)
			dongfu.POST("/gathering/start", p.DongFu.StartGathering)
			dongfu.POST("/gathering/collect", p.DongFu.CollectGathering)
			dongfu.GET("/gathering/status", p.DongFu.GetGatheringStatus)
			dongfu.POST("/decorate", p.DongFu.PlaceDecoration)
			dongfu.DELETE("/decorate/:decoration_id", p.DongFu.RemoveDecoration)
			dongfu.GET("/decorations", p.DongFu.ListDecorations)
			dongfu.POST("/guest/invite", p.DongFu.InviteGuest)
			dongfu.POST("/guest/action", p.DongFu.GuestAction)
			dongfu.GET("/guests", p.DongFu.GetGuests)
			dongfu.GET("/invitations", p.DongFu.GetInvitations)
			dongfu.GET("/passive", p.DongFu.GetPassiveRewards)
			dongfu.POST("/passive/collect", p.DongFu.CollectPassiveRewards)
		}
		v1.GET("/dongfu/thresholds", p.DongFu.GetDongFuLevelThresholds)

		pet := v1.Group("/:id/pet")
		{
			pet.GET("", p.Pet.ListPets)
			pet.GET("/:pet_id", p.Pet.GetPet)
			pet.POST("/encounter", p.Pet.TryEncounter)
			pet.POST("/capture", p.Pet.Capture)
			pet.POST("/:pet_id/rename", p.Pet.Rename)
			pet.POST("/:pet_id/feed", p.Pet.Feed)
			pet.GET("/:pet_id/level-info", p.Pet.GetLevelUpInfo)
			pet.POST("/:pet_id/evolve", p.Pet.Evolve)
			pet.POST("/:pet_id/active", p.Pet.SetActive)
			pet.POST("/:pet_id/deactivate", p.Pet.UnsetActive)
		}

		artifact := v1.Group("/:id/artifact")
		{
			artifact.POST("/bind", p.Artifact.BindArtifact)
			artifact.GET("", p.Artifact.GetArtifact)
			artifact.GET("/list", p.Artifact.ListArtifacts)
			artifact.GET("/:aid", p.Artifact.GetArtifact)
			artifact.POST("/:aid/upgrade", p.Artifact.UpgradeArtifact)
			artifact.POST("/:aid/evolve", p.Artifact.EvolveArtifact)
			artifact.GET("/:aid/awaken", p.Artifact.GetAwakenInfo)
			artifact.POST("/:aid/awaken", p.Artifact.AwakenArtifact)
			artifact.POST("/:aid/spirit", p.Artifact.ActivateSpirit)
			artifact.GET("/:aid/spirit", p.Artifact.GetSpirit)
			artifact.POST("/spirit/:sid/interact", p.Artifact.InteractSpirit)
			artifact.GET("/resonance", p.Artifact.GetResonance)
			artifact.GET("/:aid/trials", p.Artifact.GetTrialStages)
			artifact.POST("/:aid/trials/:stageId", p.Artifact.EnterTrial)
		}

		formation := v1.Group("/:id/formation")
		{
			formation.POST("/learn", p.Formation.Learn)
			formation.GET("", p.Formation.List)
			formation.POST("/deploy", p.Formation.Deploy)
			formation.POST("/undeploy", p.Formation.Undeploy)
			formation.POST("/upgrade", p.Formation.Upgrade)
			formation.POST("/guardian", p.Formation.Guardian)
			formation.POST("/unguard", p.Formation.UnsetGuardian)
			formation.GET("/guardian-history", p.Formation.GuardianHistory)
			formation.GET("/bonuses", p.Formation.DeployedBonuses)
			formation.GET("/:fid/mastery", p.Formation.MasteryInfo)
			formation.POST("/:fid/guardian-pet", p.Formation.AssignGuardian)
			formation.DELETE("/:fid/guardian-pet", p.Formation.RemoveGuardian)
			formation.POST("/:fid/link", p.Formation.SetLink)
			formation.DELETE("/:fid/link", p.Formation.ClearLink)
			formation.DELETE("/links", p.Formation.ClearAllLinks)
			formation.GET("/links/bonuses", p.Formation.LinkBonuses)
			formation.GET("/break", p.Formation.CalcBreak)
			formation.POST("/break/apply", p.Formation.ApplyBreak)
		}
		v1.GET("/formation/templates", p.Formation.Templates)

		energy := v1.Group("/:id/energy")
		{
			energy.GET("/status", p.Energy.GetStatus)
			energy.POST("/meditate", p.Energy.Meditate)
			energy.POST("/use-pill", p.Energy.UseEnergyPill)
			energy.POST("/use-potion", p.Energy.UseEnergyPill)
			energy.GET("/check/:action", p.Energy.CheckEnergy)
			energy.POST("/consume", p.Energy.ConsumeEnergy)
			energy.POST("/technique-bonus", p.Energy.SetTechniqueBonus)
		}

		rebirthRoute := v1.Group("/:id/rebirth")
		{
			rebirthRoute.GET("/check", p.Rebirth.Check)
			rebirthRoute.POST("/execute", p.Rebirth.Execute)
			rebirthRoute.GET("/benefits", p.Rebirth.Benefits)
			rebirthRoute.GET("/list", p.Rebirth.List)
			rebirthRoute.GET("/talent", p.Rebirth.GetTalentInfo)
			rebirthRoute.POST("/talent/learn", p.Rebirth.LearnTalent)
			rebirthRoute.POST("/talent/reset", p.Rebirth.ResetTalents)
			rebirthRoute.GET("/shop", p.Rebirth.GetRebirthShop)
			rebirthRoute.POST("/shop/buy", p.Rebirth.BuyRebirthShopItem)
			rebirthRoute.GET("/titles", p.Rebirth.ListTitles)
		}
	}

	// Equipment sets (separate prefix)
	equipV1 := r.Group("/api/v1/equipment")
	{
		equipV1.GET("/sets", p.EquipmentSet.ListSets)
		equipV1.GET("/sets/active/:playerID", p.EquipmentSet.GetActiveBonuses)
		equipV1.GET("/sets/progress/:playerID", p.EquipmentSet.GetSetProgress)
		equipV1.GET("/sets/missing/:playerID/:setName", p.EquipmentSet.GetMissingPieces)
		equipV1.GET("/enchants", p.EquipmentSet.GetEnchantmentList)
		equipV1.POST("/enchant", p.EquipmentSet.ApplyEnchant)
		equipV1.POST("/enchant/remove", p.EquipmentSet.RemoveEnchant)
		equipV1.GET("/:equipmentID/enchants", p.EquipmentSet.GetEquipmentEnchants)
		equipV1.POST("/awaken", p.EquipmentSet.AwakenEquipment)
		equipV1.GET("/:equipmentID/awakening", p.EquipmentSet.GetAwakeningInfo)
		equipV1.GET("/:equipmentID/can-awaken", p.EquipmentSet.CheckCanAwaken)
		equipV1.GET("/details/:equipmentID", p.EquipmentSet.GetEquipmentDetail)
	}

	// Player protection
	protectionV1 := r.Group("/api/v1/protection")
	{
		protectionV1.GET("/status", p.Protection.GetStatus)
		protectionV1.POST("/breakthrough-grace/use", p.Protection.UseBreakthroughGrace)
		protectionV1.GET("/breakthrough-grace", p.Protection.CheckBreakthroughGrace)
		protectionV1.POST("/free-resurrection/use", p.Protection.UseFreeResurrection)
	}

	// Activity / events
	activityV1 := r.Group("/api/v1/activity")
	{
		activityV1.GET("/events", p.Activity.ListEvents)
		activityV1.GET("/events/:eventID", p.Activity.GetEventDetail)
		activityV1.POST("/events/:eventID/claim", p.Activity.ClaimEventReward)
		activityV1.GET("/battlepass", p.Activity.GetBattlePass)
		activityV1.POST("/battlepass/buy", p.Activity.BuyPremiumBP)
		activityV1.POST("/battlepass/claim/:level", p.Activity.ClaimBPReward)
		activityV1.GET("/checkin/month", p.Activity.GetMonthlyCheckin)
		activityV1.POST("/checkin", p.Activity.DoCheckin)
		activityV1.POST("/checkin/makeup", p.Activity.DoMakeupCheckin)
		activityV1.GET("/achievements/:playerID", p.Activity.GetPlayerAchievements)
		activityV1.POST("/achievements/claim", p.Activity.ClaimAchievement)
		activityV1.GET("/titles/:playerID", p.Activity.GetPlayerTitles)
		activityV1.POST("/titles/equip", p.Activity.EquipTitle)
	}

	// Referral
	referralV1 := r.Group("/api/v1/referral")
	{
		referralV1.GET("/info", p.Referral.GetReferralInfo)
		referralV1.POST("/apply", p.Referral.ApplyInviteCode)
		referralV1.POST("/claim/:inviteeId", p.Referral.ClaimReferralReward)
	}

	// VIP
	vipV1 := r.Group("/api/v1/vip")
	{
		vipV1.GET("/info", p.Vip.GetVipInfo)
		vipV1.POST("/claim-daily", p.Vip.ClaimDailyReward)
		vipV1.POST("/recharge", p.Vip.ProcessRecharge)
		vipV1.GET("/recharge-history", p.Vip.GetRechargeHistory)
		vipV1.POST("/activate-monthly-card", p.Vip.ActivateMonthlyCard)
		vipV1.GET("/monthly-card-status", p.Vip.GetMonthlyCardStatus)
	}

	// ---- Combat 服务路由 ----
	c := combat
	pveGroup := r.Group("/api/v1/pve")
	{
		pveGroup.POST("/battle", c.PVE.StartBattle)
		pveGroup.GET("/monsters", c.PVE.GetMonsters)
		pveGroup.GET("/instances", c.PVE.GetInstances)
	}

	combatGroup := r.Group("/api/v1/combat")
	{
		combatGroup.GET("/monsters", c.PVE.GetMonsters)
		combatGroup.POST("/start", c.PVE.StartBattle)
		combatGroup.POST("/sweep", c.PVE.Sweep)
		combatGroup.GET("/dungeons", c.Dungeon.HandleListDungeons)
		combatGroup.POST("/dungeon/enter", c.Dungeon.HandleEnterDungeon)
		combatGroup.POST("/dungeon/fight", c.Dungeon.HandleFight)
		combatGroup.POST("/dungeon/claim", c.Dungeon.HandleClaimReward)
		combatGroup.GET("/dungeon/status", c.Dungeon.HandleStatus)
	}

	pvpGroup := r.Group("/api/v1/pvp")
	{
		pvpGroup.POST("/join", c.PVP.JoinQueue)
		pvpGroup.POST("/leave", c.PVP.LeaveQueue)
		pvpGroup.GET("/queue-status", c.PVP.QueueStatus)
		pvpGroup.POST("/action", c.PVP.SubmitAction)
		pvpGroup.GET("/status", c.PVP.GetBattleStatus)
		pvpGroup.GET("/rankings", c.PVP.GetRankings)
	}

	arenaGroup := r.Group("/api/v1/arena")
	{
		arenaGroup.POST("/match", c.Arena.HandleMatch)
		arenaGroup.GET("/status", c.Arena.HandleStatus)
		arenaGroup.GET("/rankings", c.Arena.HandleRankings)
		arenaGroup.POST("/cancel-match", c.Arena.HandleCancelMatch)
		arenaGroup.GET("/season", c.Arena.HandleSeason)
		arenaGroup.GET("/history", c.Arena.HandleHistory)
	}

	towerGroup := r.Group("/api/v1/tower")
	{
		towerGroup.POST("/enter", c.Tower.HandleEnter)
		towerGroup.POST("/fight", c.Tower.HandleFight)
		towerGroup.GET("/status", c.Tower.HandleStatus)
		towerGroup.GET("/ranking", c.Tower.HandleRanking)
	}

	c.SectDungeon.RegisterRoutes(r)
	c.TeamDungeon.RegisterRoutes(r)

	// ---- World 服务路由 ----
	w := world
	w.World.RegisterRoutes(r)
	w.Quest.RegisterRoutes(r)
	w.Vein.RegisterRoutes(r)
	w.Boss.RegisterRoutes(r)
	w.Fishing.RegisterRoutes(r)
	w.Ascend.RegisterRoutes(r)
	w.Divination.RegisterRoutes(r)
	w.Treasure.RegisterRoutes(r)

	// ---- Trade 服务路由 ----
	t := trade
	t.Handler.RegisterRoutes(r)
	t.BlackMarket.RegisterRoutes(r.Group(""))
	t.RegisterShopRoutes(r.Group(""))

	// ---- Ranking 服务路由 ----
	ranking.Handler.RegisterRoutes(r)

	// ---- Social 服务路由 ----
	if social != nil {
		s := social
		v1Social := r.Group("/api/v1")
		{
			v1Social.GET("/chat/ws", s.Chat.HandleWebSocket)
			v1Social.GET("/chat/history", s.Chat.GetHistory)
			v1Social.POST("/chat/online", s.Chat.GetOnlineStatus)
			v1Social.POST("/chat/system-notify", s.Chat.SendSystemNotification)
				v1Social.POST("/chat/send", jwtAuthMiddleware(jwtManager), s.Chat.SendMessage)

			v1Social.GET("/friends", s.Friend.GetFriendList)
			v1Social.GET("/friends/blacklist", s.Friend.GetBlacklist)
			v1Social.POST("/friends/apply", s.Friend.ApplyFriend)
			v1Social.POST("/friends/handle-apply", s.Friend.HandleApply)
			v1Social.GET("/friends/pending-applies", s.Friend.GetPendingApplies)
			v1Social.DELETE("/friends", s.Friend.RemoveFriend)
			v1Social.POST("/friends/block", s.Friend.BlockUser)
			v1Social.POST("/friends/unblock", s.Friend.UnblockUser)

			v1Social.POST("/mail/system", s.Mail.SendSystemMail)
			v1Social.POST("/mail/player", s.Mail.SendPlayerMail)
			v1Social.GET("/mail/inbox", s.Mail.GetInbox)
			v1Social.GET("/mail/read", s.Mail.ReadMail)
			v1Social.POST("/mail/claim", s.Mail.ClaimAttachment)
			v1Social.DELETE("/mail", s.Mail.DeleteMail)
			v1Social.GET("/mail/unread-count", s.Mail.CountUnread)

			v1Social.POST("/sect/create", s.Sect.CreateSect)
			v1Social.GET("/sect/my", s.Sect.GetUserSect)
			v1Social.GET("/sect/search", s.Sect.SearchSect)
			v1Social.GET("/sect/:id", s.Sect.GetSect)
			v1Social.POST("/sect/join", s.Sect.JoinSect)
			v1Social.POST("/sect/handle-apply", s.Sect.HandleApply)
			v1Social.POST("/sect/leave", s.Sect.LeaveSect)
			v1Social.POST("/sect/kick", s.Sect.KickMember)
			v1Social.POST("/sect/transfer-leader", s.Sect.TransferLeader)
			v1Social.POST("/sect/set-rank", s.Sect.SetMemberRank)
			v1Social.POST("/sect/contribution", s.Sect.AddContribution)
			v1Social.GET("/sect/contribution-rank", s.Sect.GetContributionRank)
			v1Social.GET("/sect/skills", s.Sect.GetSectSkills)
			v1Social.POST("/sect/learn-skill", s.Sect.LearnSkill)
			v1Social.GET("/sect/my-skills", s.Sect.GetMemberSkills)

			v1Social.POST("/sect/skill/list", s.SectExtra.GetSkillTree)
			v1Social.POST("/sect/skill/learn", s.SectExtra.LearnSkill)
			v1Social.POST("/sect/skill/upgrade", s.SectExtra.UpgradeSectSkill)
			v1Social.POST("/sect/mission/list", s.SectExtra.GetDailyMissions)
			v1Social.POST("/sect/mission/claim", s.SectExtra.ClaimMission)
			v1Social.POST("/sect/war/status", s.SectExtra.GetWarStatus)
			v1Social.POST("/sect/war/enroll", s.SectExtra.EnrollWar)
			v1Social.POST("/sect/rank", s.SectExtra.GetSectRank)

				// 宗门科技树
				v1Social.GET("/sect/tech/list", s.SectTech.ListTechs)
				v1Social.POST("/sect/tech/upgrade", s.SectTech.UpgradeTech)

				// 宗门仓库
				v1Social.POST("/sect/warehouse/donate", s.SectWarehouse.DonateItem)
				v1Social.GET("/sect/warehouse/list", s.SectWarehouse.GetWarehouseItems)
				v1Social.POST("/sect/warehouse/buy", s.SectWarehouse.BuyItem)
				v1Social.POST("/sect/warehouse/donate-funds", s.SectWarehouse.DonateFunds)

				// 功法阁
				v1Social.GET("/sect/technique/list", s.SectTechnique.GetTechniques)
				v1Social.POST("/sect/technique/exchange", s.SectTechnique.ExchangeTechnique)
				v1Social.POST("/sect/technique/upgrade", s.SectTechnique.UpgradeTechnique)
				v1Social.GET("/sect/technique/my", s.SectTechnique.GetMyTechniques)

			v1Social.POST("/daolv/propose", s.DaoLv.Propose)
			v1Social.POST("/daolv/handle-proposal", s.DaoLv.HandleProposal)
			v1Social.GET("/daolv/status/:playerID", s.DaoLv.GetStatus)
			v1Social.POST("/daolv/dual-cultivate", s.DaoLv.StartDualCultivate)
			v1Social.POST("/daolv/stop-cultivate", s.DaoLv.StopDualCultivate)
			v1Social.POST("/daolv/skill/:skillName", s.DaoLv.UseSkill)
			v1Social.GET("/daolv/tasks/:playerID", s.DaoLv.GetTasks)
			v1Social.POST("/daolv/task/claim", s.DaoLv.ClaimTask)
			v1Social.POST("/daolv/dissolve", s.DaoLv.Dissolve)
			v1Social.GET("/daolv/proposals/:playerID", s.DaoLv.GetProposals)
			v1Social.GET("/daolv/pending-proposals/:playerID", s.DaoLv.GetPendingProposals)

			v1Social.POST("/social/daolv/propose", s.DaoLv.OldPropose)
			v1Social.POST("/social/daolv/accept", s.DaoLv.OldAccept)
			v1Social.POST("/social/daolv/reject", s.DaoLv.OldReject)
			v1Social.POST("/social/daolv/divorce", s.DaoLv.OldDivorce)
			v1Social.POST("/social/daolv/cultivate", s.DaoLv.OldDualCultivate)
			v1Social.POST("/social/daolv/gift", s.DaoLv.OldSendGift)
			v1Social.POST("/social/daolv/teleport", s.DaoLv.OldTeleport)
			v1Social.GET("/social/daolv/info", s.DaoLv.OldGetInfo)
			v1Social.GET("/social/daolv/proposals", s.DaoLv.OldGetProposals)

			v1Social.POST("/master/apply", s.Master.Apply)
			v1Social.POST("/master/accept", s.Master.Accept)
			v1Social.POST("/master/reject", s.Master.Reject)
			v1Social.GET("/master/pending-applies", s.Master.GetPendingApplies)
			v1Social.GET("/master/my-master", s.Master.GetMyMaster)
			v1Social.GET("/master/my-students", s.Master.GetMyStudents)
			v1Social.POST("/master/teach", s.Master.Teach)
			v1Social.GET("/master/missions", s.Master.GetDailyMissions)
			v1Social.POST("/master/mission/progress", s.Master.UpdateMissionProgress)
			v1Social.POST("/master/mission/claim", s.Master.ClaimMission)
			v1Social.POST("/master/graduate", s.Master.Graduate)
			v1Social.POST("/master/kick", s.Master.Kick)
			v1Social.GET("/master/master-value", s.Master.GetMasterValue)
			v1Social.GET("/master/level", s.Master.GetMentorshipLevelInfo)
			v1Social.POST("/master/level/upgrade", s.Master.UpgradeMentorshipLevel)
			v1Social.POST("/master/training/assign", s.Master.AssignDailyTraining)
			v1Social.GET("/master/training", s.Master.GetDailyTraining)
			v1Social.POST("/master/training/progress", s.Master.UpdateTrainingProgress)
			v1Social.POST("/master/training/claim", s.Master.ClaimTrainingReward)
			v1Social.POST("/master/betray", s.Master.Betray)
			v1Social.GET("/master/betray-history", s.Master.GetBetrayalHistory)
			v1Social.POST("/master/dungeon/create", s.Master.CreateDungeon)
			v1Social.POST("/master/dungeon/enter", s.Master.EnterDungeon)
			v1Social.POST("/master/dungeon/wave-complete", s.Master.DungeonWaveComplete)
			v1Social.POST("/master/dungeon/claim", s.Master.ClaimDungeonReward)
			v1Social.GET("/master/dungeon/status", s.Master.GetDungeonInstance)
		}
	} else {
		slog.Warn("social routes skipped (MongoDB not available)")
	}

	// ---- Cultivation 路由 (http.ServeMux, via NoRoute) ----
	r.NoRoute(func(c *gin.Context) {
		cultivation.Mux.ServeHTTP(c.Writer, c.Request)
	})

	// ---------------------------------------------------------------
	// 15. 启动 HTTP 服务器
	// ---------------------------------------------------------------
	addr := ":" + serverPort
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("unified server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// ---------------------------------------------------------------
	// 16. 等待信号，优雅关闭
	// ---------------------------------------------------------------
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("shutting down", "signal", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	if mongoClient != nil {
		if err := mongoClient.Disconnect(shutdownCtx); err != nil {
			slog.Warn("MongoDB disconnect error", "error", err)
		}
	}

	slog.Info("unified server stopped")
}


// ============================================================================
// Middleware
// ============================================================================

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()
		slog.Info("request",
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency", time.Since(start),
			"remote", c.Request.RemoteAddr,
		)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// ============================================================================
// Auth schema initialization
// ============================================================================

func initAuthSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '用户唯一 ID',
		username VARCHAR(64) NOT NULL COMMENT '用户名',
		password_hash VARCHAR(255) NOT NULL COMMENT '密码哈希',
		player_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '关联的玩家角色 ID',
		email VARCHAR(128) NOT NULL DEFAULT '' COMMENT '电子邮箱',
		status TINYINT NOT NULL DEFAULT 0 COMMENT '账号状态：0=正常 1=封禁 2=冻结 3=已删除',
		last_login_at DATETIME NULL COMMENT '最后登录时间',
		last_ip VARCHAR(64) NOT NULL DEFAULT '' COMMENT '最后登录 IP',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
		UNIQUE KEY uk_username (username),
		INDEX idx_player_id (player_id),
		INDEX idx_status (status)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户账号表';`
	if _, err := appDB.ExecContext(context.Background(), schema); err != nil {
		return err
	}

	gmTables := []string{
		`CREATE TABLE IF NOT EXISTS gm_admins (
			id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '管理员唯一ID',
			username VARCHAR(64) NOT NULL COMMENT '管理员用户名',
			password_hash VARCHAR(255) NOT NULL COMMENT '密码哈希',
			role TINYINT NOT NULL DEFAULT 3 COMMENT '角色：1=超级管理员 2=运营 3=观察者',
			status TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1=启用 0=禁用',
			last_login_at TIMESTAMP NULL DEFAULT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY uk_username (username),
			INDEX idx_role (role),
			INDEX idx_status (status)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='GM 管理员账号表'`,
		`CREATE TABLE IF NOT EXISTS gm_operation_logs (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			admin_id BIGINT NOT NULL,
			action_type VARCHAR(64) NOT NULL,
			target_type VARCHAR(64) NOT NULL DEFAULT '',
			target_id BIGINT NOT NULL DEFAULT 0,
			detail JSON DEFAULT NULL,
			ip_address VARCHAR(45) NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_admin_id (admin_id),
			INDEX idx_action_type (action_type),
			INDEX idx_created_at (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='GM 操作日志表'`,
		`CREATE TABLE IF NOT EXISTS gm_announcements (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			admin_id BIGINT NOT NULL,
			title VARCHAR(128) NOT NULL,
			content TEXT NOT NULL,
			type TINYINT NOT NULL DEFAULT 1,
			target_player_id BIGINT NULL DEFAULT NULL,
			sent_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expire_at TIMESTAMP NULL DEFAULT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_admin_id (admin_id),
			INDEX idx_type (type),
			INDEX idx_sent_at (sent_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='GM 公告表'`,
		`CREATE TABLE IF NOT EXISTS gm_bans (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			player_id BIGINT NOT NULL,
			admin_id BIGINT NOT NULL,
			reason TEXT NOT NULL,
			ban_type TINYINT NOT NULL DEFAULT 1,
			starts_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			ends_at TIMESTAMP NULL DEFAULT NULL,
			status TINYINT NOT NULL DEFAULT 1,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_player_id (player_id),
			INDEX idx_admin_id (admin_id),
			INDEX idx_status (status)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='GM 封禁记录表'`,
	}
	for _, ddl := range gmTables {
		if _, err := appDB.ExecContext(context.Background(), ddl); err != nil {
			return fmt.Errorf("init GM schema: %w", err)
		}
	}
	return nil
}

// ============================================================================
// WebSocket 升级器
// ============================================================================
// ============================================================================
// WebSocket 升级器
// ============================================================================

// 聊天枢纽
type chatClient struct {
	PlayerID uint64
	Nickname string
	Conn     *websocket.Conn
	Mu       sync.Mutex
}
var chatHub = struct {
	sync.RWMutex
	clients map[uint64]*chatClient
}{clients: make(map[uint64]*chatClient)}

func broadcastChat(msg map[string]interface{}) {
	data, _ := json.Marshal(msg)
	chatHub.RLock()
	defer chatHub.RUnlock()
	for _, c := range chatHub.clients {
		c.Mu.Lock()
		c.Conn.WriteMessage(websocket.TextMessage, data)
		c.Mu.Unlock()
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// jwtAuthMiddleware 从 Authorization header 提取并验证 JWT，将 player_id 写入 context。
func jwtAuthMiddleware(jwtManager *gatewayadapter.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "请先登录"})
			return
		}
		token := authHeader[7:]
		claims, err := jwtManager.ValidateAccessToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "token无效或已过期"})
			return
		}
		c.Set("player_id", int64(claims.PlayerID))
		c.Set("account", claims.Account)
		c.Next()
	}
}

func wsHandler(jwtManager *gatewayadapter.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "missing token"})
			return
		}
		claims, err := jwtManager.ValidateAccessToken(token)
		if err != nil {
			slog.Warn("ws auth failed", "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "invalid token"})
			return
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			slog.Error("websocket upgrade failed", "error", err)
			return
		}
		client := &chatClient{PlayerID: claims.PlayerID, Nickname: claims.Account, Conn: conn}
		chatHub.Lock()
		chatHub.clients[claims.PlayerID] = client
		chatHub.Unlock()
		var nick string
		appDB.QueryRow("SELECT nickname FROM players WHERE id=?", claims.PlayerID).Scan(&nick)
		if nick != "" { client.Nickname = nick }
		slog.Info("new websocket connection", "player_id", claims.PlayerID, "nickname", client.Nickname)
		connMsg, _ := json.Marshal(map[string]interface{}{"type":"connected","player_id":claims.PlayerID,"nickname":client.Nickname})
		conn.WriteMessage(websocket.TextMessage, connMsg)
		broadcastChat(map[string]interface{}{"type":"chat","channel":"world","name":"系统","text":client.Nickname+" 进入了修仙世界","color":"#4caf50"})
		go func() {
			defer func() {
				chatHub.Lock()
				delete(chatHub.clients, claims.PlayerID)
				chatHub.Unlock()
				broadcastChat(map[string]interface{}{"type":"chat","channel":"world","name":"系统","text":client.Nickname+" 离开了修仙世界","color":"#888"})
				conn.Close()
			}()
			for {
				_, msgBytes, err := conn.ReadMessage()
				if err != nil { return }
				var m struct {
					Type    string `json:"type"`
					Channel string `json:"channel"`
					Text    string `json:"text"`
				}
				if json.Unmarshal(msgBytes, &m) != nil { continue }
				if m.Type == "chat" && m.Text != "" {
					broadcastChat(map[string]interface{}{
						"type":"chat","channel":m.Channel,"name":client.Nickname,
						"text":m.Text,"color":"#d4a843",
					})
				}
			}
		}()
	}
}
func loginHandler(jwtManager *gatewayadapter.JWTManager, auth *authadapter.Components) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Account  string `json:"account"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid body"})
			return
		}
		if req.Account == "" || req.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "account and password required"})
			return
		}

		result := auth.Login(c.Request.Context(), req.Account, req.Password)
		if result.Err != nil {
			slog.Warn("login failed", "account", req.Account, "error", result.Err)
			slog.Warn("login failed", "error", result.Err); c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "账号或密码错误"})
			return
		}

		gtwAccessToken, gtwRefreshToken, err := jwtManager.GenerateTokenPair(result.PlayerID, req.Account)
		if err != nil {
			slog.Error("generate token pair error", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "internal error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"player_id":     result.PlayerID,
			"user_id":       result.UserID,
			"account":       req.Account,
			"access_token":  gtwAccessToken,
			"refresh_token": gtwRefreshToken,
		})
	}
}

func registerHandler(jwtManager *gatewayadapter.JWTManager, auth *authadapter.Components) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Account  string `json:"account"`
			Password string `json:"password"`
			Nickname string `json:"nickname"`
			Gender   string `json:"gender"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid body"})
			return
		}
		if req.Account == "" || req.Password == "" || req.Nickname == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "account, password and nickname required"})
			return
		}
		if req.Gender == "" {
			req.Gender = "male"
		}

		result := auth.Register(c.Request.Context(), req.Account, req.Password, req.Nickname, req.Gender)
		if result.Err != nil {
			slog.Warn("register failed", "account", req.Account, "error", result.Err)
			slog.Warn("register failed", "error", result.Err); c.JSON(http.StatusConflict, gin.H{"code": 409, "msg": "账号已存在，请换一个"})
			return
		}

		gtwAccessToken, gtwRefreshToken, err := jwtManager.GenerateTokenPair(result.PlayerID, req.Account)
		if err != nil {
			slog.Error("generate token pair error", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "internal error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"player_id":     result.PlayerID,
			"user_id":       result.UserID,
			"account":       req.Account,
			"access_token":  gtwAccessToken,
			"refresh_token": gtwRefreshToken,
		})
	}
}

func refreshTokenHandler(jwtManager *gatewayadapter.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid body"})
			return
		}
		if req.RefreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "refresh_token is required"})
			return
		}

		accessToken, err := jwtManager.RefreshAccessToken(req.RefreshToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "invalid refresh token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"access_token":  accessToken,
			"refresh_token": req.RefreshToken,
		})
	}
}

// ============================================================================
// Env helpers
// ============================================================================

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
