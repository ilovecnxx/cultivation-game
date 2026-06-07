// Package combatadapter wraps combat service internals for monolith consumption.
package combatadapter

import (
	"path/filepath"

	"cultivation-game/services/combat/internal/config"
	"cultivation-game/services/combat/internal/handler"
	"cultivation-game/services/combat/internal/repository"
	"cultivation-game/services/combat/internal/service"
)

// Handlers holds all initialized combat service handlers.
type Handlers struct {
	PVE            *handler.PVEHandler
	PVP            *handler.PVPHandler
	Arena          *handler.ArenaHandler
	Dungeon        *handler.DungeonHandler
	Tower          *handler.TowerHandler
	SectDungeon    *handler.SectDungeonHandler
	TeamDungeon    *handler.TeamDungeonHandler
	SectDungeonSvc *service.SectDungeonService
	TeamDungeonSvc *service.TeamDungeonService
}

// Bootstrap initializes the combat service layer and returns handlers.
// dataDir is the path to the combat service's internal/data/ directory.
func Bootstrap(dataDir, playerAddr string) *Handlers {
	cfg := config.DefaultConfig()
	cfg.DataPath.Monsters = filepath.Join(dataDir, "monsters.json")
	cfg.DataPath.Skills = filepath.Join(dataDir, "skills.json")
	cfg.DataPath.Instances = filepath.Join(dataDir, "instances.json")
	cfg.DataPath.Dungeons = filepath.Join(dataDir, "dungeons.json")

	playerClient := repository.NewPlayerClient(playerAddr)

	h := &Handlers{
		PVE:     handler.NewPVEHandler(cfg),
		PVP:     handler.NewPVPHandler(cfg, playerClient),
		Arena:   handler.NewArenaHandler(cfg),
		Dungeon: handler.NewDungeonHandler(cfg),
		Tower:   handler.NewTowerHandler(cfg),
	}

	h.SectDungeonSvc = service.NewSectDungeonService()
	h.SectDungeonSvc.Start()
	h.SectDungeon = handler.NewSectDungeonHandler(h.SectDungeonSvc)

	h.TeamDungeonSvc = service.NewTeamDungeonService()
	h.TeamDungeonSvc.StartCleanupTask()
	h.TeamDungeon = handler.NewTeamDungeonHandler(h.TeamDungeonSvc)

	// Load dungeon data
	if err := h.Dungeon.LoadDungeonData(cfg.DataPath.Dungeons); err != nil {
		println("failed to load dungeon data:", err.Error())
	}

	return h
}

// Stop cleans up background goroutines.
func (h *Handlers) Stop() {
	if h.SectDungeonSvc != nil {
		h.SectDungeonSvc.Stop()
	}
}
