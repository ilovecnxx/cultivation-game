// Package socialadapter wraps social service internals for monolith consumption.
package socialadapter

import (
	"log/slog"

	"cultivation-game/services/social/internal/handler"
	"cultivation-game/services/social/internal/repository"
	"cultivation-game/services/social/internal/service"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

// Handlers holds all initialized social service handlers.
type Handlers struct {
	Chat        *handler.ChatHandler
	Friend      *handler.FriendHandler
	Mail        *handler.MailHandler
	Sect        *handler.SectHandler
	SectExtra   *handler.SectExtraHandler
	DaoLv       *handler.DaoLvHandler
	Master      *handler.MasterHandler
}

// Bootstrap initializes the social service layer.
// Returns nil if MongoDB is not available.
func Bootstrap(mongoDB *mongo.Database, rdb *redis.Client, playerAddr string) *Handlers {
	if mongoDB == nil {
		return nil
	}
	log := slog.Default()

	chatRepo := repository.NewChatRepo(mongoDB)
	mailRepo := repository.NewMailRepo(mongoDB)
	chatSvc := service.NewChatService(chatRepo, rdb)
	friendSvc := service.NewFriendService(mongoDB, rdb)
	mailSvc := service.NewMailService(mailRepo, friendSvc)
	sectSvc := service.NewSectService(mongoDB)
	sectSkillSvc := service.NewSectSkillService(mongoDB)
	sectMissionSvc := service.NewSectMissionService(mongoDB)
	sectWarSvc := service.NewSectWarService(log, mongoDB)
	daolvRepo := repository.NewDaoLvRepo(mongoDB)
	daolvSvc := service.NewDaoLvService(daolvRepo, mongoDB)
	playerClient := repository.NewPlayerClient(playerAddr)
	masterSvc := service.NewMasterService(mongoDB, playerClient)

	return &Handlers{
		Chat:      handler.NewChatHandler(log, chatSvc),
		Friend:    handler.NewFriendHandler(friendSvc),
		Mail:      handler.NewMailHandler(mailSvc),
		Sect:      handler.NewSectHandler(sectSvc),
		SectExtra: handler.NewSectExtraHandler(sectSkillSvc, sectMissionSvc, sectWarSvc),
		DaoLv:     handler.NewDaoLvHandler(daolvSvc),
		Master:    handler.NewMasterHandler(masterSvc),
	}
}
