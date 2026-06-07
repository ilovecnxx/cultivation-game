// Package worldadapter wraps world service internals for monolith consumption.
package worldadapter

import (
	"log/slog"
	"path/filepath"

	"cultivation-game/services/world/internal/handler"
	"cultivation-game/services/world/internal/service"

	"github.com/redis/go-redis/v9"
)

// Handlers holds all initialized world service handlers.
type Handlers struct {
	World       *handler.WorldHandler
	Quest       *handler.QuestHandler
	Vein        *handler.VeinHandler
	Boss        *handler.WorldBossHandler
	Fishing     *handler.FishingHandler
	Ascend      *handler.AscendHandler
	Divination  *handler.DivinationHandler
	Treasure    *handler.TreasureHandler
	BossSvc     *service.WorldBossService
}

// Bootstrap initializes the world service layer.
// dataDir is the path to the world service's internal/data/ directory.
func Bootstrap(rdb *redis.Client, dataDir string) *Handlers {
	defaultRegionsPath := filepath.Join(dataDir, "map_regions.json")
	defaultNPCsPath := filepath.Join(dataDir, "npcs.json")
	defaultSpotsPath := filepath.Join(dataDir, "gathering_spots.json")
	defaultEncountersPath := filepath.Join(dataDir, "encounters.json")
	defaultQuestsPath := filepath.Join(dataDir, "quests.json")
	defaultFishingPath := filepath.Join(dataDir, "fishing_spots.json")
	defaultDailyTasksPath := filepath.Join(dataDir, "daily_tasks.json")

	exploreSvc, err := service.NewExploreService(
		defaultRegionsPath, defaultNPCsPath, defaultSpotsPath, rdb, dataDir)
	if err != nil {
		slog.Warn("failed to create explore service", "error", err)
	}

	encounterSvc, err := service.NewEncounterService(defaultEncountersPath, exploreSvc)
	if err != nil {
		slog.Warn("failed to create encounter service", "error", err)
	}

	questSvc, err := service.NewQuestService(defaultQuestsPath)
	if err != nil {
		slog.Warn("failed to create quest service", "error", err)
	}
	if err := questSvc.LoadDailyTasks(defaultDailyTasksPath); err != nil {
		slog.Warn("failed to load daily tasks", "error", err)
	}

	spiritDensitySvc, err := service.NewSpiritDensityService(defaultRegionsPath, 50)
	if err != nil {
		slog.Warn("failed to create spirit density service", "error", err)
	}

	veinSvc, err := service.NewSpiritVeinService(rdb, dataDir)
	if err != nil {
		slog.Warn("failed to create spirit vein service", "error", err)
	}

	fishingSvc, err := service.NewFishingService(defaultFishingPath)
	if err != nil {
		slog.Warn("failed to create fishing service", "error", err)
	}

	bossSvc := service.NewWorldBossService()
	bossSvc.Start()

	divinationSvc := service.NewDivinationService()
	treasureSvc := service.NewTreasureService()

	return &Handlers{
		World:      handler.NewWorldHandler(exploreSvc, encounterSvc, spiritDensitySvc),
		Quest:      handler.NewQuestHandler(questSvc),
		Vein:       handler.NewVeinHandler(veinSvc),
		Boss:       handler.NewWorldBossHandler(bossSvc),
		Fishing:    handler.NewFishingHandler(fishingSvc),
		Ascend:     handler.NewAscendHandler(dataDir),
		Divination: handler.NewDivinationHandler(divinationSvc),
		Treasure:   handler.NewTreasureHandler(treasureSvc),
		BossSvc:    bossSvc,
	}
}
