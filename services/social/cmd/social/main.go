// Social 社交服务 - 修仙游戏微服务
//
// 功能:
//   - 聊天系统(世界/宗门/私聊/系统)
//   - 好友系统(添加/删除/黑名单)
//   - 邮件系统(系统/玩家邮件, 附件)
//   - 宗门系统(创建/管理/技能/宗门战)
//   - 双修系统(双人修炼)
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cultivation-game/services/social/internal/config"
	"cultivation-game/services/social/internal/handler"
	"cultivation-game/services/social/internal/repository"
	"cultivation-game/services/social/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	configPath := flag.String("config", "config.json", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	// 连接 MongoDB
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.MongoDB.URI))
	if err != nil {
		logger.Error("连接 MongoDB 失败", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := mongoClient.Disconnect(context.Background()); err != nil {
			logger.Warn("关闭 MongoDB 连接失败", "error", err)
		}
	}()
	mongoDB := mongoClient.Database(cfg.MongoDB.Database)

	// 连接 Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		logger.Error("连接 Redis 失败", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	// 初始化仓储层
	chatRepo := repository.NewChatRepo(mongoDB)
	mailRepo := repository.NewMailRepo(mongoDB)

	// 初始化服务层
	chatSvc := service.NewChatService(chatRepo, rdb)
	friendSvc := service.NewFriendService(mongoDB, rdb)
	mailSvc := service.NewMailService(mailRepo, friendSvc)
	sectSvc := service.NewSectService(mongoDB)
	sectSkillSvc := service.NewSectSkillService(mongoDB)
	sectMissionSvc := service.NewSectMissionService(mongoDB)
	sectWarSvc := service.NewSectWarService(logger, mongoDB)

	// 道侣系统
	daolvRepo := repository.NewDaoLvRepo(mongoDB)
	daolvSvc := service.NewDaoLvService(daolvRepo, mongoDB)

	// 师徒系统
	playerClient := repository.NewPlayerClient(cfg.Services.PlayerServiceAddr)
	masterSvc := service.NewMasterService(mongoDB, playerClient)
	// 注意: MasterService 的 GetPlayerRealm 和 AddPlayerExp 
	// 接口由 PlayerClient 实现,通过 HTTP 调用 player 服务
	// 增加 GetPlayerExp 和 GetPlayerAttrs 方法支持
	masterHandler := handler.NewMasterHandler(masterSvc)

	// 初始化 HTTP 处理器
	chatHandler := handler.NewChatHandler(logger, chatSvc)
	friendHandler := handler.NewFriendHandler(friendSvc)
	mailHandler := handler.NewMailHandler(mailSvc)
	sectHandler := handler.NewSectHandler(sectSvc)
	sectExtraHandler := handler.NewSectExtraHandler(sectSkillSvc, sectMissionSvc, sectWarSvc)
	daolvHandler := handler.NewDaoLvHandler(daolvSvc)

	// 设置 Gin 路由
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// 健康检查（含 Redis 和 MongoDB 状态）
	router.GET("/health", func(c *gin.Context) {
		mongoStatus := "ok"
		if err := mongoClient.Ping(context.Background(), nil); err != nil {
			mongoStatus = "degraded: " + err.Error()
		}
		redisStatus := "ok"
		if err := rdb.Ping(context.Background()).Err(); err != nil {
			redisStatus = "degraded: " + err.Error()
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "social",
			"mongodb": mongoStatus,
			"redis":   redisStatus,
		})
	})

	// API 路由
	v1 := router.Group("/api/v1")
	{
		// 聊天
		v1.GET("/chat/ws", chatHandler.HandleWebSocket)
		v1.GET("/chat/history", chatHandler.GetHistory)
		v1.POST("/chat/online", chatHandler.GetOnlineStatus)
		v1.POST("/chat/system-notify", chatHandler.SendSystemNotification)

		// 好友
		v1.GET("/friends", friendHandler.GetFriendList)
		v1.GET("/friends/blacklist", friendHandler.GetBlacklist)
		v1.POST("/friends/apply", friendHandler.ApplyFriend)
		v1.POST("/friends/handle-apply", friendHandler.HandleApply)
		v1.GET("/friends/pending-applies", friendHandler.GetPendingApplies)
		v1.DELETE("/friends", friendHandler.RemoveFriend)
		v1.POST("/friends/block", friendHandler.BlockUser)
		v1.POST("/friends/unblock", friendHandler.UnblockUser)

		// 邮件
		v1.POST("/mail/system", mailHandler.SendSystemMail)
		v1.POST("/mail/player", mailHandler.SendPlayerMail)
		v1.GET("/mail/inbox", mailHandler.GetInbox)
		v1.GET("/mail/read", mailHandler.ReadMail)
		v1.POST("/mail/claim", mailHandler.ClaimAttachment)
		v1.DELETE("/mail", mailHandler.DeleteMail)
		v1.GET("/mail/unread-count", mailHandler.CountUnread)

		// 宗门
		v1.POST("/sect/create", sectHandler.CreateSect)
		v1.GET("/sect/my", sectHandler.GetUserSect)
		v1.GET("/sect/search", sectHandler.SearchSect)
		v1.GET("/sect/:id", sectHandler.GetSect)
		v1.POST("/sect/join", sectHandler.JoinSect)
		v1.POST("/sect/handle-apply", sectHandler.HandleApply)
		v1.POST("/sect/leave", sectHandler.LeaveSect)
		v1.POST("/sect/kick", sectHandler.KickMember)
		v1.POST("/sect/transfer-leader", sectHandler.TransferLeader)
		v1.POST("/sect/set-rank", sectHandler.SetMemberRank)
		v1.POST("/sect/contribution", sectHandler.AddContribution)
		v1.GET("/sect/contribution-rank", sectHandler.GetContributionRank)
		v1.GET("/sect/skills", sectHandler.GetSectSkills)
		v1.POST("/sect/learn-skill", sectHandler.LearnSkill)
		v1.GET("/sect/my-skills", sectHandler.GetMemberSkills)

		// 宗门技能扩展
		v1.POST("/sect/skill/list", sectExtraHandler.GetSkillTree)
		v1.POST("/sect/skill/learn", sectExtraHandler.LearnSkill)
		v1.POST("/sect/skill/upgrade", sectExtraHandler.UpgradeSectSkill)

		// 宗门任务
		v1.POST("/sect/mission/list", sectExtraHandler.GetDailyMissions)
		v1.POST("/sect/mission/claim", sectExtraHandler.ClaimMission)

		// 宗门战
		v1.POST("/sect/war/status", sectExtraHandler.GetWarStatus)
		v1.POST("/sect/war/enroll", sectExtraHandler.EnrollWar)

		// 宗门排名
		v1.POST("/sect/rank", sectExtraHandler.GetSectRank)

		// 道侣系统
		v1.POST("/daolv/propose", daolvHandler.Propose)
		v1.POST("/daolv/handle-proposal", daolvHandler.HandleProposal)
		v1.GET("/daolv/status/:playerID", daolvHandler.GetStatus)
		v1.POST("/daolv/dual-cultivate", daolvHandler.StartDualCultivate)
		v1.POST("/daolv/stop-cultivate", daolvHandler.StopDualCultivate)
		v1.POST("/daolv/skill/:skillName", daolvHandler.UseSkill)
		v1.GET("/daolv/tasks/:playerID", daolvHandler.GetTasks)
		v1.POST("/daolv/task/claim", daolvHandler.ClaimTask)
		v1.POST("/daolv/dissolve", daolvHandler.Dissolve)
		v1.GET("/daolv/proposals/:playerID", daolvHandler.GetProposals)
		v1.GET("/daolv/pending-proposals/:playerID", daolvHandler.GetPendingProposals)

		// 道侣系统(旧接口兼容)
		v1.POST("/social/daolv/propose", daolvHandler.OldPropose)
		v1.POST("/social/daolv/accept", daolvHandler.OldAccept)
		v1.POST("/social/daolv/reject", daolvHandler.OldReject)
		v1.POST("/social/daolv/divorce", daolvHandler.OldDivorce)
		v1.POST("/social/daolv/cultivate", daolvHandler.OldDualCultivate)
		v1.POST("/social/daolv/gift", daolvHandler.OldSendGift)
		v1.POST("/social/daolv/teleport", daolvHandler.OldTeleport)
		v1.GET("/social/daolv/info", daolvHandler.OldGetInfo)
		v1.GET("/social/daolv/proposals", daolvHandler.OldGetProposals)

		// 师徒系统
		v1.POST("/master/apply", masterHandler.Apply)
		v1.POST("/master/accept", masterHandler.Accept)
		v1.POST("/master/reject", masterHandler.Reject)
		v1.GET("/master/pending-applies", masterHandler.GetPendingApplies)
		v1.GET("/master/my-master", masterHandler.GetMyMaster)
		v1.GET("/master/my-students", masterHandler.GetMyStudents)
		v1.POST("/master/teach", masterHandler.Teach)
		v1.GET("/master/missions", masterHandler.GetDailyMissions)
		v1.POST("/master/mission/progress", masterHandler.UpdateMissionProgress)
		v1.POST("/master/mission/claim", masterHandler.ClaimMission)
		v1.POST("/master/graduate", masterHandler.Graduate)
		v1.POST("/master/kick", masterHandler.Kick)
		v1.GET("/master/master-value", masterHandler.GetMasterValue)
		// 师徒等级
		v1.GET("/master/level", masterHandler.GetMentorshipLevelInfo)
		v1.POST("/master/level/upgrade", masterHandler.UpgradeMentorshipLevel)
		// 每日训练
		v1.POST("/master/training/assign", masterHandler.AssignDailyTraining)
		v1.GET("/master/training", masterHandler.GetDailyTraining)
		v1.POST("/master/training/progress", masterHandler.UpdateTrainingProgress)
		v1.POST("/master/training/claim", masterHandler.ClaimTrainingReward)
		// 叛离师门
		v1.POST("/master/betray", masterHandler.Betray)
		v1.GET("/master/betray-history", masterHandler.GetBetrayalHistory)
		// 师徒副本
		v1.POST("/master/dungeon/create", masterHandler.CreateDungeon)
		v1.POST("/master/dungeon/enter", masterHandler.EnterDungeon)
		v1.POST("/master/dungeon/wave-complete", masterHandler.DungeonWaveComplete)
		v1.POST("/master/dungeon/claim", masterHandler.ClaimDungeonReward)
		v1.GET("/master/dungeon/status", masterHandler.GetDungeonInstance)
	}

	// 启动 HTTP 服务
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// 优雅关闭
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		logger.Info("正在关闭社交服务")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("服务关闭失败", "error", err)
		os.Exit(1)
		}
	}()

	logger.Info("社交服务启动", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("服务启动失败", "error", err)
		os.Exit(1)
	}
	logger.Info("社交服务已关闭")
}
