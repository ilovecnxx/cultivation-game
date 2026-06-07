// Package playeradapter wraps player service internals for monolith consumption.
package playeradapter

import (
	"database/sql"
	"log/slog"
	"path/filepath"

	"cultivation-game/services/player/internal/handler"
	mysqlRepo "cultivation-game/services/player/internal/repository/mysql"
	redisRepo "cultivation-game/services/player/internal/repository/redis"
	"cultivation-game/services/player/internal/service"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Handlers holds all initialized player service handlers.
type Handlers struct {
	Player       *handler.PlayerHandler
	Protection   *handler.ProtectionHandler
	Equipment    *handler.EquipmentHandler
	DongFu       *handler.DongFuHandler
	Pet          *handler.PetHandler
	Formation    *handler.FormationHandler
	Artifact     *handler.ArtifactHandler
	Rebirth      *handler.RebirthHandler
	Energy       *handler.EnergyHandler
	EquipmentSet *handler.EquipmentSetHandler
	Activity     *handler.ActivityHandler
	Referral     *handler.ReferralHandler
	Vip          *handler.VipHandler
	Cultivation  *handler.CultivationHandler
	Training     *handler.TrainingHandler
	Pill         *handler.PillHandler
	Friend       *handler.FriendHandler
	Pve          *handler.PveHandler
	Backpack     *handler.BackpackHandler
}

// Bootstrap initializes the player service layer and returns handlers.
// dataDir is the path to the player service's internal/data/ directory.
func Bootstrap(db *sql.DB, rdb *redis.Client, dataDir string) *Handlers {
	log, _ := zap.NewProduction()

	// ---- Repos ----
	playerRepo := mysqlRepo.NewPlayerRepo(db, log)
	inventoryRepo := mysqlRepo.NewInventoryRepo(db, log)
	artifactRepo := mysqlRepo.NewArtifactRepo(db, log)
	dongfuRepo := mysqlRepo.NewDongFuRepo(db, log)
	petRepo := mysqlRepo.NewPetRepo(db, log)
	formationRepo := mysqlRepo.NewFormationRepo(db, log)
	cache := redisRepo.NewCache(rdb, log)
	energyRepo := mysqlRepo.NewEnergyRepo(db, log)
	protectionRepo := mysqlRepo.NewProtectionRepo(db, log)
	checkinRepo := mysqlRepo.NewCheckinRepo(db, log)
	referralRepo := mysqlRepo.NewReferralRepo(db, log)
	activityRepo := mysqlRepo.NewActivityRepo(db, log)
	vipRepo := mysqlRepo.NewVipRepo(db, log)

	// ---- Services ----
	playerSvc := service.NewPlayerService(playerRepo, cache, log)
	inventorySvc := service.NewInventoryService(inventoryRepo, playerSvc, cache, log)
	artifactSvc := service.NewArtifactService(artifactRepo, playerRepo, inventoryRepo, log)
	dongfuSvc := service.NewDongFuService(dongfuRepo, playerRepo, log)
	petSvc := service.NewPetService(petRepo, playerRepo, inventorySvc, log)
	formationSvc := service.NewFormationService(formationRepo, playerRepo, petRepo, log)
	energySvc := service.NewEnergyService(energyRepo, playerSvc, log)
	protectionSvc := service.NewProtectionService(protectionRepo, playerRepo, log)
	rebirthSvc := service.NewRebirthService(db, playerRepo, dongfuRepo, artifactRepo, log)
	equipmentSetSvc := service.NewEquipmentSetService(inventoryRepo, log)
	activitySvc := service.NewActivityService(activityRepo, checkinRepo, playerSvc, inventorySvc, log)
	referralSvc := service.NewReferralService(referralRepo, playerSvc, inventorySvc, log)
	vipSvc := service.NewVipService(vipRepo, playerSvc, inventorySvc, log)

	// Load data files
	formationsPath := filepath.Join(dataDir, "formations.json")
	if err := formationSvc.LoadTemplates(formationsPath); err != nil {
		slog.Warn("failed to load formation templates", "error", err)
	}
	energyConfigPath := filepath.Join(dataDir, "energy.json")
	if err := energySvc.LoadConfig(energyConfigPath); err != nil {
		slog.Warn("failed to load energy config", "error", err)
	}

	// ---- Handlers ----
	return &Handlers{
		Player:       handler.NewPlayerHandler(playerSvc, inventorySvc, log),
		Protection:   handler.NewProtectionHandler(protectionSvc, log),
		DongFu:       handler.NewDongFuHandler(dongfuSvc, log),
		Pet:          handler.NewPetHandler(petSvc, log),
		Formation:    handler.NewFormationHandler(formationSvc, log),
		Artifact:     handler.NewArtifactHandler(artifactSvc, log),
		Rebirth:      handler.NewRebirthHandler(rebirthSvc, log),
		Energy:       handler.NewEnergyHandler(energySvc, log),
		EquipmentSet: handler.NewEquipmentSetHandler(equipmentSetSvc, inventorySvc, log),
		Activity:     handler.NewActivityHandler(activitySvc, log),
		Referral:     handler.NewReferralHandler(referralSvc, log),
		Vip:          handler.NewVipHandler(vipSvc, log),
		Cultivation:  handler.NewCultivationHandler(playerSvc, log),
		Training:     handler.NewTrainingHandler(playerSvc, log),
		Pill:         handler.NewPillHandler(db, playerSvc, log),
		Friend:       handler.NewFriendHandler(db, playerSvc, log),
		Pve:          handler.NewPveHandler(playerSvc, log),
		Backpack:     handler.NewBackpackHandler(db, playerSvc, log),
	}
}
