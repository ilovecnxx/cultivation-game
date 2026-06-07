package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"cultivation-game/services/player/internal/config"
	"cultivation-game/services/player/internal/handler"
	mysqlRepo "cultivation-game/services/player/internal/repository/mysql"
	redisRepo "cultivation-game/services/player/internal/repository/redis"
	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化日志
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer logger.Sync()

	// 连接 MySQL
	db, err := sql.Open("mysql", cfg.MySQL.DSN)
	if err != nil {
		logger.Fatal("打开数据库连接失败", zap.Error(err))
	}
	defer db.Close()

	db.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.MySQL.ConnMaxLifetime)

	if err := db.Ping(); err != nil {
		logger.Fatal("Ping 数据库失败", zap.Error(err))
	}
	logger.Info("MySQL 连接成功")

	// 连接 Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		logger.Fatal("Ping Redis 失败", zap.Error(err))
	}
	logger.Info("Redis 连接成功")

	// 初始化 Repository 层
	playerRepo := mysqlRepo.NewPlayerRepo(db, logger)
	inventoryRepo := mysqlRepo.NewInventoryRepo(db, logger)
	artifactRepo := mysqlRepo.NewArtifactRepo(db, logger)
	dongfuRepo := mysqlRepo.NewDongFuRepo(db, logger)
	petRepo := mysqlRepo.NewPetRepo(db, logger)
	formationRepo := mysqlRepo.NewFormationRepo(db, logger)
	cache := redisRepo.NewCache(rdb, logger)

	// 初始化 Service 层
	playerService := service.NewPlayerService(playerRepo, cache, logger)
	inventoryService := service.NewInventoryService(inventoryRepo, playerService, cache, logger)
	artifactService := service.NewArtifactService(artifactRepo, playerRepo, inventoryRepo, logger)
	dongfuService := service.NewDongFuService(dongfuRepo, playerRepo, logger)
	petService := service.NewPetService(petRepo, playerRepo, inventoryService, logger)
	formationService := service.NewFormationService(formationRepo, playerRepo, petRepo, logger)

	// 加载阵法图谱
	formationsPath := filepath.Join("..", "..", "internal", "data", "formations.json")
	if err := formationService.LoadTemplates(formationsPath); err != nil {
		logger.Fatal("加载阵法图谱失败", zap.Error(err))
	}

	// 初始化能量/体力服务
	energyRepo := mysqlRepo.NewEnergyRepo(db, logger)
	energyService := service.NewEnergyService(energyRepo, playerService, logger)
	energyConfigPath := filepath.Join("..", "..", "internal", "data", "energy.json")
	if err := energyService.LoadConfig(energyConfigPath); err != nil {
		logger.Fatal("加载能量配置失败", zap.Error(err))
	}

	// 初始化新手保护服务
	protectionRepo := mysqlRepo.NewProtectionRepo(db, logger)
	protectionService := service.NewProtectionService(protectionRepo, playerRepo, logger)

	// 初始化轮回服务
	rebirthService := service.NewRebirthService(db, playerRepo, dongfuRepo, artifactRepo, logger)

	// 初始化能量服务处理器
	energyHandler := handler.NewEnergyHandler(energyService, logger)

	// 初始化装备套装服务
	equipmentSetService := service.NewEquipmentSetService(inventoryRepo, logger)
	equipmentSetHandler := handler.NewEquipmentSetHandler(equipmentSetService, inventoryService, logger)

	// 签到仓库
	checkinRepo := mysqlRepo.NewCheckinRepo(db, logger)

	// 推荐系统仓库
	referralRepo := mysqlRepo.NewReferralRepo(db, logger)

	// ================= 运营活动框架 =================
	activityRepo := mysqlRepo.NewActivityRepo(db, logger)
	activityService := service.NewActivityService(activityRepo, checkinRepo, playerService, inventoryService, logger)
	activityHandler := handler.NewActivityHandler(activityService, logger)

	// 推荐系统
	referralService := service.NewReferralService(referralRepo, playerService, inventoryService, logger)
	referralHandler := handler.NewReferralHandler(referralService, logger)

	// ================= VIP系统 =================
	vipRepo := mysqlRepo.NewVipRepo(db, logger)
	vipService := service.NewVipService(vipRepo, playerService, inventoryService, logger)
	vipHandler := handler.NewVipHandler(vipService, logger)

	// 初始化 Handler 层
	playerHandler := handler.NewPlayerHandler(playerService, inventoryService, logger)
	protectionHandler := handler.NewProtectionHandler(protectionService, logger)
	inventoryHandler := handler.NewInventoryHandler(inventoryService, logger)
	equipmentHandler := handler.NewEquipmentHandler(inventoryService, logger)
	dongfuHandler := handler.NewDongFuHandler(dongfuService, logger)
	petHandler := handler.NewPetHandler(petService, logger)
	formationHandler := handler.NewFormationHandler(formationService, logger)
	artifactHandler := handler.NewArtifactHandler(artifactService, logger)
	rebirthHandler := handler.NewRebirthHandler(rebirthService, logger)

	r := gin.Default()

	// 健康检查（含 MySQL 和 Redis 状态）
	r.GET("/health", func(c *gin.Context) {
		dbStatus := "ok"
		if err := db.Ping(); err != nil {
			dbStatus = "degraded: " + err.Error()
		}
		redisStatus := "ok"
		if err := rdb.Ping(context.Background()).Err(); err != nil {
			redisStatus = "degraded: " + err.Error()
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "player",
			"mysql":   dbStatus,
			"redis":   redisStatus,
		})
	})

	// 玩家服务 API
	v1 := r.Group("/api/v1/player")
	{
		// 玩家 CRUD
		v1.POST("/register", playerHandler.Register)
		v1.GET("/:id", playerHandler.GetProfile)
		v1.PUT("/:id", playerHandler.UpdateProfile)
		v1.GET("/user/:user_id", playerHandler.GetByUserID)
		v1.POST("/:id/currency", playerHandler.UpdateCurrency)

		// 跨服务调用接口（供其他服务 HTTP 调用）
		v1.POST("/:id/update-realm", playerHandler.UpdateRealm)
		v1.POST("/:id/add-exp", playerHandler.AddExp)
		v1.POST("/:id/update-exp", playerHandler.UpdateExp)
		v1.POST("/:id/update-attributes", playerHandler.UpdateAttributes)
		v1.POST("/:id/add-item", playerHandler.AddItem)
		v1.POST("/:id/remove-item", playerHandler.RemoveItem)
		v1.GET("/:id/attributes", playerHandler.GetAttributes)

		// 背包操作
		inventory := v1.Group("/:id/inventory")
		{
			inventory.GET("", inventoryHandler.ListInventory)
			inventory.POST("/add", inventoryHandler.AddItem)
			inventory.POST("/remove", inventoryHandler.RemoveItem)
			inventory.POST("/transfer", inventoryHandler.TransferItem)
			inventory.POST("/sort", inventoryHandler.SortInventory)
			inventory.POST("/use", inventoryHandler.UseItem)
		}

		// 装备操作
		equipment := v1.Group("/:id/equipment")
		{
			equipment.GET("", equipmentHandler.ListEquipment)
			equipment.POST("/equip", equipmentHandler.Equip)
			equipment.POST("/unequip", equipmentHandler.Unequip)
			equipment.POST("/strengthen", equipmentHandler.Strengthen)
		}

		// 洞府操作
		dongfu := v1.Group("/:id/dongfu")
		{
			dongfu.POST("/build", dongfuHandler.Build)
			dongfu.GET("", dongfuHandler.GetDongFu)
			// 房间操作
			dongfu.POST("/room/build", dongfuHandler.BuildRoom)
			dongfu.POST("/room/upgrade", dongfuHandler.UpgradeRoom)
			dongfu.GET("/room/:room_id", dongfuHandler.GetRoomDetail)
			// 灵气汇聚
			dongfu.POST("/gathering/start", dongfuHandler.StartGathering)
			dongfu.POST("/gathering/collect", dongfuHandler.CollectGathering)
			dongfu.GET("/gathering/status", dongfuHandler.GetGatheringStatus)
			// 装饰系统
			dongfu.POST("/decorate", dongfuHandler.PlaceDecoration)
			dongfu.DELETE("/decorate/:decoration_id", dongfuHandler.RemoveDecoration)
			dongfu.GET("/decorations", dongfuHandler.ListDecorations)
			// 访客系统
			dongfu.POST("/guest/invite", dongfuHandler.InviteGuest)
			dongfu.POST("/guest/action", dongfuHandler.GuestAction)
			dongfu.GET("/guests", dongfuHandler.GetGuests)
			dongfu.GET("/invitations", dongfuHandler.GetInvitations)
			// 被动收益
			dongfu.GET("/passive", dongfuHandler.GetPassiveRewards)
			dongfu.POST("/passive/collect", dongfuHandler.CollectPassiveRewards)
		}
		// 洞府配置（无玩家ID）
		v1.GET("/dongfu/thresholds", dongfuHandler.GetDongFuLevelThresholds)

		// 灵兽操作
		pet := v1.Group("/:id/pet")
		{
			pet.GET("", petHandler.ListPets)
			pet.GET("/:pet_id", petHandler.GetPet)
			pet.POST("/encounter", petHandler.TryEncounter)
			pet.POST("/capture", petHandler.Capture)
			pet.POST("/:pet_id/rename", petHandler.Rename)
			pet.POST("/:pet_id/feed", petHandler.Feed)
			pet.GET("/:pet_id/level-info", petHandler.GetLevelUpInfo)
			pet.POST("/:pet_id/evolve", petHandler.Evolve)
			pet.POST("/:pet_id/active", petHandler.SetActive)
			pet.POST("/:pet_id/deactivate", petHandler.UnsetActive)
		}

		// 法宝操作
		artifact := v1.Group("/:id/artifact")
		{
			artifact.POST("/bind", artifactHandler.BindArtifact)
			artifact.GET("", artifactHandler.GetArtifact)
			artifact.GET("/list", artifactHandler.ListArtifacts)
			artifact.GET("/:aid", artifactHandler.GetArtifact)
			artifact.POST("/:aid/upgrade", artifactHandler.UpgradeArtifact)
			artifact.POST("/:aid/evolve", artifactHandler.EvolveArtifact)
			artifact.GET("/:aid/awaken", artifactHandler.GetAwakenInfo)
			artifact.POST("/:aid/awaken", artifactHandler.AwakenArtifact)
			artifact.POST("/:aid/spirit", artifactHandler.ActivateSpirit)
			artifact.GET("/:aid/spirit", artifactHandler.GetSpirit)
			artifact.POST("/spirit/:sid/interact", artifactHandler.InteractSpirit)
			artifact.GET("/resonance", artifactHandler.GetResonance)
			artifact.GET("/:aid/trials", artifactHandler.GetTrialStages)
			artifact.POST("/:aid/trials/:stageId", artifactHandler.EnterTrial)
		}
		// 阵法操作
		formation := v1.Group("/:id/formation")
		{
			formation.POST("/learn", formationHandler.Learn)
			formation.GET("", formationHandler.List)
			formation.POST("/deploy", formationHandler.Deploy)
			formation.POST("/undeploy", formationHandler.Undeploy)
			formation.POST("/upgrade", formationHandler.Upgrade)
			formation.POST("/guardian", formationHandler.Guardian)
			formation.POST("/unguard", formationHandler.UnsetGuardian)
			formation.GET("/guardian-history", formationHandler.GuardianHistory)
			formation.GET("/bonuses", formationHandler.DeployedBonuses)

			// --- v2 增强 ---
			// 熟练度
			formation.GET("/:fid/mastery", formationHandler.MasteryInfo)

			// 守护灵兽
			formation.POST("/:fid/guardian-pet", formationHandler.AssignGuardian)
			formation.DELETE("/:fid/guardian-pet", formationHandler.RemoveGuardian)

			// 联动系统
			formation.POST("/:fid/link", formationHandler.SetLink)
			formation.DELETE("/:fid/link", formationHandler.ClearLink)
			formation.DELETE("/links", formationHandler.ClearAllLinks)
			formation.GET("/links/bonuses", formationHandler.LinkBonuses)

			// 破阵（PVP）
			formation.GET("/break", formationHandler.CalcBreak)
			formation.POST("/break/apply", formationHandler.ApplyBreak)
		}

		// 阵法图谱（无需玩家ID）
		v1.GET("/formation/templates", formationHandler.Templates)

		// 体力/能量系统 (v2.0 - 修炼打坐恢复)
		energy := v1.Group("/:id/energy")
		{
			energy.GET("/status", energyHandler.GetStatus)
			// 修炼打坐恢复体力
			energy.POST("/meditate", energyHandler.Meditate)
			// 体力丹药恢复（多品阶）
			energy.POST("/use-pill", energyHandler.UseEnergyPill)
			// 兼容旧版药水接口
			energy.POST("/use-potion", energyHandler.UseEnergyPill)
			// 检查/消耗
			energy.GET("/check/:action", energyHandler.CheckEnergy)
			energy.POST("/consume", energyHandler.ConsumeEnergy)
			// 功法体力回复加成（由修炼服务调用）
			energy.POST("/technique-bonus", energyHandler.SetTechniqueBonus)
		}

		// 轮回转世系统
		rebirthRoute := v1.Group("/:id/rebirth")
		{
			rebirthRoute.GET("/check", rebirthHandler.Check)
			rebirthRoute.POST("/execute", rebirthHandler.Execute)
			rebirthRoute.GET("/benefits", rebirthHandler.Benefits)
			rebirthRoute.GET("/list", rebirthHandler.List)

			// 天赋树
			rebirthRoute.GET("/talent", rebirthHandler.GetTalentInfo)
			rebirthRoute.POST("/talent/learn", rebirthHandler.LearnTalent)
			rebirthRoute.POST("/talent/reset", rebirthHandler.ResetTalents)

			// 轮回商店
			rebirthRoute.GET("/shop", rebirthHandler.GetRebirthShop)
			rebirthRoute.POST("/shop/buy", rebirthHandler.BuyRebirthShopItem)

			// 称号
			rebirthRoute.GET("/titles", rebirthHandler.ListTitles)
		}

		// 装备套装/附魔/觉醒 API（独立路由，非玩家ID绑定）
		equipV1 := r.Group("/api/v1/equipment")
		{
			equipV1.GET("/sets", equipmentSetHandler.ListSets)
			equipV1.GET("/sets/active/:playerID", equipmentSetHandler.GetActiveBonuses)
			equipV1.GET("/sets/progress/:playerID", equipmentSetHandler.GetSetProgress)
			equipV1.GET("/sets/missing/:playerID/:setName", equipmentSetHandler.GetMissingPieces)
			equipV1.GET("/enchants", equipmentSetHandler.GetEnchantmentList)
			equipV1.POST("/enchant", equipmentSetHandler.ApplyEnchant)
			equipV1.POST("/enchant/remove", equipmentSetHandler.RemoveEnchant)
			equipV1.GET("/:equipmentID/enchants", equipmentSetHandler.GetEquipmentEnchants)
			equipV1.POST("/awaken", equipmentSetHandler.AwakenEquipment)
			equipV1.GET("/:equipmentID/awakening", equipmentSetHandler.GetAwakeningInfo)
			equipV1.GET("/:equipmentID/can-awaken", equipmentSetHandler.CheckCanAwaken)
			equipV1.GET("/details/:equipmentID", equipmentSetHandler.GetEquipmentDetail)
		}
	}

	// ================= 新手保护 API =================
	protectionV1 := r.Group("/api/v1/protection")
	{
		protectionV1.GET("/status", protectionHandler.GetStatus)
		protectionV1.POST("/breakthrough-grace/use", protectionHandler.UseBreakthroughGrace)
		protectionV1.GET("/breakthrough-grace", protectionHandler.CheckBreakthroughGrace)
		protectionV1.POST("/free-resurrection/use", protectionHandler.UseFreeResurrection)
	}

	// ================= 运营活动 API =================
	activityV1 := r.Group("/api/v1/activity")
	{
		// 限时活动
		activityV1.GET("/events", activityHandler.ListEvents)
		activityV1.GET("/events/:eventID", activityHandler.GetEventDetail)
		activityV1.POST("/events/:eventID/claim", activityHandler.ClaimEventReward)

		// 战令
		activityV1.GET("/battlepass", activityHandler.GetBattlePass)
		activityV1.POST("/battlepass/buy", activityHandler.BuyPremiumBP)
		activityV1.POST("/battlepass/claim/:level", activityHandler.ClaimBPReward)

		// 签到增强
		activityV1.GET("/checkin/month", activityHandler.GetMonthlyCheckin)
		activityV1.POST("/checkin", activityHandler.DoCheckin)
		activityV1.POST("/checkin/makeup", activityHandler.DoMakeupCheckin)

		// 成就增强
		activityV1.GET("/achievements/:playerID", activityHandler.GetPlayerAchievements)
		activityV1.POST("/achievements/claim", activityHandler.ClaimAchievement)

		// 称号系统
		activityV1.GET("/titles/:playerID", activityHandler.GetPlayerTitles)
		activityV1.POST("/titles/equip", activityHandler.EquipTitle)
	}

		// ================= 推荐/邀请系统 API =================
		referralV1 := r.Group("/api/v1/referral")
		{
			referralV1.GET("/info", referralHandler.GetReferralInfo)
			referralV1.POST("/apply", referralHandler.ApplyInviteCode)
			referralV1.POST("/claim/:inviteeId", referralHandler.ClaimReferralReward)
		}

	// ================= VIP系统 API =================
	vipV1 := r.Group("/api/v1/vip")
	{
		vipV1.GET("/info", vipHandler.GetVipInfo)
		vipV1.POST("/claim-daily", vipHandler.ClaimDailyReward)
		vipV1.POST("/recharge", vipHandler.ProcessRecharge)
		vipV1.GET("/recharge-history", vipHandler.GetRechargeHistory)
		vipV1.POST("/activate-monthly-card", vipHandler.ActivateMonthlyCard)
		vipV1.GET("/monthly-card-status", vipHandler.GetMonthlyCardStatus)
	}

	// HTTP 服务
	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// 优雅关闭
	go func() {
		logger.Info("玩家服务启动", zap.String("addr", cfg.Server.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("服务启动失败", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("服务关闭失败", zap.Error(err))
	}
	logger.Info("服务已关闭")
}
