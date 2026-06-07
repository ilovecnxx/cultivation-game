package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"cultivation-game/services/ranking/internal/config"
	"cultivation-game/services/ranking/internal/model"
)

// ============================================================================
// Mock RankingRepository
// ============================================================================

type mockRankingRepo struct {
	updateScoreFn       func(ctx context.Context, rankingType model.RankingType, playerID uint64, score float64, nickname, realmName string) error
	getTopNFn           func(ctx context.Context, rankingType model.RankingType, n int64) ([]*model.RankingEntry, error)
	getRankingByPageFn  func(ctx context.Context, rankingType model.RankingType, page *model.PageRequest) ([]*model.RankingEntry, int64, error)
	getPlayerRankFn     func(ctx context.Context, rankingType model.RankingType, playerID uint64) (int64, float64, error)
	getPlayerInfoFn     func(ctx context.Context, rankingType model.RankingType, playerID uint64) (*model.RankingEntry, error)
	getNeighborsFn      func(ctx context.Context, rankingType model.RankingType, playerID uint64, neighborCount int64) (above, below []*model.RankingEntry, err error)
	batchUpdateScoresFn func(ctx context.Context, rankingType model.RankingType, entries []*model.RankingEntry) error
	updateActivityFn    func(ctx context.Context, rankingType model.RankingType, playerID uint64) error
	setSnapshotFn       func(ctx context.Context, rankingType model.RankingType, entries []*model.RankingEntry, ttl time.Duration) error
	getInactivePlayersFn func(ctx context.Context, rankingType model.RankingType, deadline time.Time) ([]uint64, error)
	applyDecayToPlayerFn func(ctx context.Context, rankingType model.RankingType, playerID uint64, decayRate float64) error
}

func (m *mockRankingRepo) UpdateScore(ctx context.Context, rankingType model.RankingType, playerID uint64, score float64, nickname, realmName string) error {
	if m.updateScoreFn != nil {
		return m.updateScoreFn(ctx, rankingType, playerID, score, nickname, realmName)
	}
	return nil
}
func (m *mockRankingRepo) GetTopN(ctx context.Context, rankingType model.RankingType, n int64) ([]*model.RankingEntry, error) {
	if m.getTopNFn != nil {
		return m.getTopNFn(ctx, rankingType, n)
	}
	return nil, nil
}
func (m *mockRankingRepo) GetRankingByPage(ctx context.Context, rankingType model.RankingType, page *model.PageRequest) ([]*model.RankingEntry, int64, error) {
	if m.getRankingByPageFn != nil {
		return m.getRankingByPageFn(ctx, rankingType, page)
	}
	return nil, 0, nil
}
func (m *mockRankingRepo) GetPlayerRank(ctx context.Context, rankingType model.RankingType, playerID uint64) (int64, float64, error) {
	if m.getPlayerRankFn != nil {
		return m.getPlayerRankFn(ctx, rankingType, playerID)
	}
	return -1, 0, nil
}
func (m *mockRankingRepo) GetPlayerInfo(ctx context.Context, rankingType model.RankingType, playerID uint64) (*model.RankingEntry, error) {
	if m.getPlayerInfoFn != nil {
		return m.getPlayerInfoFn(ctx, rankingType, playerID)
	}
	return nil, nil
}
func (m *mockRankingRepo) GetNeighbors(ctx context.Context, rankingType model.RankingType, playerID uint64, neighborCount int64) (above, below []*model.RankingEntry, err error) {
	if m.getNeighborsFn != nil {
		return m.getNeighborsFn(ctx, rankingType, playerID, neighborCount)
	}
	return []*model.RankingEntry{}, []*model.RankingEntry{}, nil
}
func (m *mockRankingRepo) BatchUpdateScores(ctx context.Context, rankingType model.RankingType, entries []*model.RankingEntry) error {
	if m.batchUpdateScoresFn != nil {
		return m.batchUpdateScoresFn(ctx, rankingType, entries)
	}
	return nil
}
func (m *mockRankingRepo) UpdateActivity(ctx context.Context, rankingType model.RankingType, playerID uint64) error {
	if m.updateActivityFn != nil {
		return m.updateActivityFn(ctx, rankingType, playerID)
	}
	return nil
}
func (m *mockRankingRepo) SetSnapshot(ctx context.Context, rankingType model.RankingType, entries []*model.RankingEntry, ttl time.Duration) error {
	if m.setSnapshotFn != nil {
		return m.setSnapshotFn(ctx, rankingType, entries, ttl)
	}
	return nil
}
func (m *mockRankingRepo) GetInactivePlayers(ctx context.Context, rankingType model.RankingType, deadline time.Time) ([]uint64, error) {
	if m.getInactivePlayersFn != nil {
		return m.getInactivePlayersFn(ctx, rankingType, deadline)
	}
	return nil, nil
}
func (m *mockRankingRepo) ApplyDecayToPlayer(ctx context.Context, rankingType model.RankingType, playerID uint64, decayRate float64) error {
	if m.applyDecayToPlayerFn != nil {
		return m.applyDecayToPlayerFn(ctx, rankingType, playerID, decayRate)
	}
	return nil
}

// ============================================================================
// Test helpers
// ============================================================================

func newTestConfig() *config.Config {
	return &config.Config{
		UpdateBufferSize:     100,
		UpdateWorkerCount:    2,
		CacheRefreshInterval: 30 * time.Second,
		CacheTopN:            100,
		DecayCheckInterval:   10 * time.Minute,
	}
}

func newTestRankingService(repo RankingRepository, cfg *config.Config) *RankingService {
	// When testing, we create the service without the background workers to avoid goroutine leaks.
	// We only test synchronous public API methods.
	return &RankingService{
		repo:      repo,
		cfg:       cfg,
		log:       slog.Default(),
		updateCh:  make(chan *UpdateTask, cfg.UpdateBufferSize),
		snapshots: make(map[model.RankingType]*model.Snapshot),
		stopCh:    make(chan struct{}),
	}
}

// ============================================================================
// UpdateScore
// ============================================================================

func TestUpdateScore(t *testing.T) {
	ctx := context.Background()

	t.Run("success enqueue", func(t *testing.T) {
		svc := newTestRankingService(&mockRankingRepo{}, newTestConfig())
		err := svc.UpdateScore(ctx, model.RankingTypeCombatPower, 1001, 50000, "剑仙", "筑基三层")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("invalid ranking type", func(t *testing.T) {
		svc := newTestRankingService(&mockRankingRepo{}, newTestConfig())
		err := svc.UpdateScore(ctx, "invalid", 1001, 50000, "test", "test")
		if err == nil {
			t.Fatal("expected error for invalid ranking type")
		}
		if err != ErrInvalidRankingType {
			t.Errorf("expected ErrInvalidRankingType, got %v", err)
		}
	})

	t.Run("new player score", func(t *testing.T) {
		var capturedScore float64
		repo := &mockRankingRepo{
			updateScoreFn: func(_ context.Context, _ model.RankingType, _ uint64, score float64, _, _ string) error {
				capturedScore = score
				return nil
			},
		}
		svc := newTestRankingService(repo, newTestConfig())
		_ = svc.UpdateScore(ctx, model.RankingTypeWealth, 2001, 99999, "财神", "金丹")
		// Allow the async worker to process
		task := <-svc.updateCh
		// Process the task manually (the worker would do this)
		_ = repo.UpdateScore(ctx, task.RankingType, task.PlayerID, task.Score, task.Nickname, task.RealmName)
		if capturedScore != 99999 {
			t.Errorf("expected score 99999, got %v", capturedScore)
		}
	})

	t.Run("channel full returns error", func(t *testing.T) {
		svc := newTestRankingService(&mockRankingRepo{}, &config.Config{
			UpdateBufferSize:  1,
			UpdateWorkerCount: 0,
		})
		// Fill the channel
		svc.updateCh <- &UpdateTask{PlayerID: 1}
		// Next send should fail
		err := svc.UpdateScore(ctx, model.RankingTypeRealm, 2, 100, "n", "n")
		if err == nil {
			t.Error("expected error when channel is full")
		}
	})
}

// ============================================================================
// GetTopPlayers
// ============================================================================

func TestGetTopPlayers(t *testing.T) {
	ctx := context.Background()

	makeEntries := func(n int) []*model.RankingEntry {
		entries := make([]*model.RankingEntry, n)
		for i := 0; i < n; i++ {
			entries[i] = &model.RankingEntry{PlayerID: uint64(i + 1), Score: float64(100 - i), Rank: int32(i + 1)}
		}
		return entries
	}

	t.Run("basic top 3", func(t *testing.T) {
		repo := &mockRankingRepo{
			getTopNFn: func(_ context.Context, _ model.RankingType, n int64) ([]*model.RankingEntry, error) {
				return makeEntries(int(n)), nil
			},
		}
		svc := newTestRankingService(repo, newTestConfig())
		// first call: cache miss, query repo
		entries, err := svc.GetTopPlayers(ctx, model.RankingTypeRealm, 3)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 3 {
			t.Errorf("expected 3 entries, got %d", len(entries))
		}
		if entries[0].Rank != 1 {
			t.Errorf("first entry Rank = %d; want 1", entries[0].Rank)
		}
	})

	t.Run("empty ranking", func(t *testing.T) {
		repo := &mockRankingRepo{
			getTopNFn: func(_ context.Context, _ model.RankingType, _ int64) ([]*model.RankingEntry, error) {
				return []*model.RankingEntry{}, nil
			},
		}
		svc := newTestRankingService(repo, newTestConfig())
		entries, err := svc.GetTopPlayers(ctx, model.RankingTypeRealm, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Errorf("expected 0 entries, got %d", len(entries))
		}
	})

	t.Run("limit clamping min", func(t *testing.T) {
		repo := &mockRankingRepo{
			getTopNFn: func(_ context.Context, _ model.RankingType, n int64) ([]*model.RankingEntry, error) {
				if n != 10 {
					t.Errorf("expected requested n=10, got %d", n)
				}
				return makeEntries(10), nil
			},
		}
		svc := newTestRankingService(repo, newTestConfig())
		// limit < 1 should be clamped to 10
		entries, err := svc.GetTopPlayers(ctx, model.RankingTypeRealm, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 10 {
			t.Errorf("expected 10 entries, got %d", len(entries))
		}
	})

	t.Run("limit clamping max", func(t *testing.T) {
		cfg := newTestConfig()
		cfg.CacheTopN = 50
		repo := &mockRankingRepo{
			getTopNFn: func(_ context.Context, _ model.RankingType, n int64) ([]*model.RankingEntry, error) {
				if n != 50 {
					t.Errorf("expected n=50 (clamped), got %d", n)
				}
				return makeEntries(50), nil
			},
		}
		svc := newTestRankingService(repo, cfg)
		// limit > CacheTopN should be clamped to CacheTopN
		entries, err := svc.GetTopPlayers(ctx, model.RankingTypeRealm, 999)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 50 {
			t.Errorf("expected 50 entries, got %d", len(entries))
		}
	})

	t.Run("invalid ranking type", func(t *testing.T) {
		svc := newTestRankingService(&mockRankingRepo{}, newTestConfig())
		_, err := svc.GetTopPlayers(ctx, "invalid", 10)
		if err != ErrInvalidRankingType {
			t.Errorf("expected ErrInvalidRankingType, got %v", err)
		}
	})

	t.Run("cache hit", func(t *testing.T) {
		callCount := 0
		repo := &mockRankingRepo{
			getTopNFn: func(_ context.Context, _ model.RankingType, _ int64) ([]*model.RankingEntry, error) {
				callCount++
				return makeEntries(100), nil
			},
		}
		svc := newTestRankingService(repo, newTestConfig())

		// First call: cache miss -> queries repo
		_, _ = svc.GetTopPlayers(ctx, model.RankingTypeRealm, 5)
		if callCount != 1 {
			t.Errorf("expected 1 repo call (miss), got %d", callCount)
		}

		// Second call: cache hit -> no repo call
		entries, err := svc.GetTopPlayers(ctx, model.RankingTypeRealm, 5)
		if err != nil {
			t.Fatal(err)
		}
		if callCount != 1 {
			t.Errorf("expected 1 repo call total (cache hit), got %d", callCount)
		}
		if len(entries) != 5 {
			t.Errorf("expected 5 entries, got %d", len(entries))
		}
	})
}

// ============================================================================
// GetPlayerRank
// ============================================================================

func TestGetPlayerRank(t *testing.T) {
	ctx := context.Background()

	t.Run("player found with neighbors", func(t *testing.T) {
		repo := &mockRankingRepo{
			getPlayerRankFn: func(_ context.Context, _ model.RankingType, _ uint64) (int64, float64, error) {
				return 5, 85000, nil // rank 5, score 85000
			},
			getPlayerInfoFn: func(_ context.Context, _ model.RankingType, _ uint64) (*model.RankingEntry, error) {
				return &model.RankingEntry{Nickname: "剑客", RealmName: "金丹"}, nil
			},
			getNeighborsFn: func(_ context.Context, _ model.RankingType, _ uint64, _ int64) ([]*model.RankingEntry, []*model.RankingEntry, error) {
				above := []*model.RankingEntry{
					{PlayerID: 1, Score: 90000, Rank: 4},
					{PlayerID: 2, Score: 88000, Rank: 3},
				}
				below := []*model.RankingEntry{
					{PlayerID: 3, Score: 82000, Rank: 6},
					{PlayerID: 4, Score: 80000, Rank: 7},
				}
				return above, below, nil
			},
		}
		svc := newTestRankingService(repo, newTestConfig())
		entry, above, below, err := svc.GetPlayerRank(ctx, model.RankingTypeCombatPower, 100)
		if err != nil {
			t.Fatal(err)
		}
		if entry == nil {
			t.Fatal("expected non-nil entry")
		}
		if entry.Rank != 5 {
			t.Errorf("Rank = %d; want 5", entry.Rank)
		}
		if entry.Score != 85000 {
			t.Errorf("Score = %v; want 85000", entry.Score)
		}
		if entry.Nickname != "剑客" {
			t.Errorf("Nickname = %s; want 剑客", entry.Nickname)
		}
		if len(above) != 2 {
			t.Errorf("expected 2 above neighbors, got %d", len(above))
		}
		if len(below) != 2 {
			t.Errorf("expected 2 below neighbors, got %d", len(below))
		}
	})

	t.Run("player not found", func(t *testing.T) {
		repo := &mockRankingRepo{
			getPlayerRankFn: func(_ context.Context, _ model.RankingType, _ uint64) (int64, float64, error) {
				return -1, 0, nil
			},
		}
		svc := newTestRankingService(repo, newTestConfig())
		_, _, _, err := svc.GetPlayerRank(ctx, model.RankingTypeRealm, 999)
		if err != ErrPlayerNotFound {
			t.Errorf("expected ErrPlayerNotFound, got %v", err)
		}
	})

	t.Run("player info fetch failure degrades gracefully", func(t *testing.T) {
		repo := &mockRankingRepo{
			getPlayerRankFn: func(_ context.Context, _ model.RankingType, _ uint64) (int64, float64, error) {
				return 3, 70000, nil
			},
			getPlayerInfoFn: func(_ context.Context, _ model.RankingType, _ uint64) (*model.RankingEntry, error) {
				return nil, errors.New("redis error")
			},
			getNeighborsFn: func(_ context.Context, _ model.RankingType, _ uint64, _ int64) ([]*model.RankingEntry, []*model.RankingEntry, error) {
				return []*model.RankingEntry{}, []*model.RankingEntry{}, nil
			},
		}
		svc := newTestRankingService(repo, newTestConfig())
		entry, _, _, err := svc.GetPlayerRank(ctx, model.RankingTypeRealm, 50)
		if err != nil {
			t.Fatal(err)
		}
		if entry.Nickname != "" {
			t.Errorf("expected empty nickname when info fetch fails, got %q", entry.Nickname)
		}
	})

	t.Run("invalid ranking type", func(t *testing.T) {
		svc := newTestRankingService(&mockRankingRepo{}, newTestConfig())
		_, _, _, err := svc.GetPlayerRank(ctx, "badtype", 1)
		if err != ErrInvalidRankingType {
			t.Errorf("expected ErrInvalidRankingType, got %v", err)
		}
	})

	t.Run("neighbors fetch failure degrades gracefully", func(t *testing.T) {
		repo := &mockRankingRepo{
			getPlayerRankFn: func(_ context.Context, _ model.RankingType, _ uint64) (int64, float64, error) {
				return 10, 50000, nil
			},
			getPlayerInfoFn: func(_ context.Context, _ model.RankingType, _ uint64) (*model.RankingEntry, error) {
				return &model.RankingEntry{Nickname: "test"}, nil
			},
			getNeighborsFn: func(_ context.Context, _ model.RankingType, _ uint64, _ int64) ([]*model.RankingEntry, []*model.RankingEntry, error) {
				return nil, nil, errors.New("redis error")
			},
		}
		svc := newTestRankingService(repo, newTestConfig())
		entry, above, below, err := svc.GetPlayerRank(ctx, model.RankingTypeRealm, 10)
		if err != nil {
			t.Fatal(err)
		}
		if entry == nil {
			t.Fatal("expected entry despite neighbor fetch error")
		}
		if above == nil || below == nil {
			t.Fatal("neighbors should be empty slices, not nil")
		}
	})
}

// ============================================================================
// BatchUpdate
// ============================================================================

func TestBatchUpdate(t *testing.T) {
	ctx := context.Background()

	t.Run("successful batch", func(t *testing.T) {
		var captured []*model.RankingEntry
		repo := &mockRankingRepo{
			batchUpdateScoresFn: func(_ context.Context, _ model.RankingType, entries []*model.RankingEntry) error {
				captured = entries
				return nil
			},
		}
		svc := newTestRankingService(repo, newTestConfig())
		entries := []*model.RankingEntry{
			{PlayerID: 1, Score: 100},
			{PlayerID: 2, Score: 90},
		}
		err := svc.BatchUpdate(ctx, model.RankingTypeWealth, entries)
		if err != nil {
			t.Fatal(err)
		}
		if len(captured) != 2 {
			t.Errorf("expected 2 entries sent to repo, got %d", len(captured))
		}
	})

	t.Run("empty batch is no-op", func(t *testing.T) {
		called := false
		repo := &mockRankingRepo{
			batchUpdateScoresFn: func(_ context.Context, _ model.RankingType, _ []*model.RankingEntry) error {
				called = true
				return nil
			},
		}
		svc := newTestRankingService(repo, newTestConfig())
		err := svc.BatchUpdate(ctx, model.RankingTypeRealm, []*model.RankingEntry{})
		if err != nil {
			t.Fatal(err)
		}
		if called {
			t.Error("batch should not call repo for empty entries")
		}
	})

	t.Run("invalid ranking type", func(t *testing.T) {
		svc := newTestRankingService(&mockRankingRepo{}, newTestConfig())
		err := svc.BatchUpdate(ctx, "bad", []*model.RankingEntry{{PlayerID: 1}})
		if err != ErrInvalidRankingType {
			t.Errorf("expected ErrInvalidRankingType, got %v", err)
		}
	})
}

// ============================================================================
// GetRanking (pagination)
// ============================================================================

func TestGetRanking(t *testing.T) {
	ctx := context.Background()

	t.Run("paginated results", func(t *testing.T) {
		repo := &mockRankingRepo{
			getRankingByPageFn: func(_ context.Context, _ model.RankingType, page *model.PageRequest) ([]*model.RankingEntry, int64, error) {
				// Verify normalization was applied
				if page.Page != 2 {
					t.Errorf("expected page 2, got %d", page.Page)
				}
				if page.PageSize != 10 {
					t.Errorf("expected pageSize 10, got %d", page.PageSize)
				}
				entries := make([]*model.RankingEntry, 10)
				for i := 0; i < 10; i++ {
					entries[i] = &model.RankingEntry{PlayerID: uint64(i + 11), Score: float64(90 - i), Rank: int32(10 + i + 1)}
				}
				return entries, 100, nil
			},
		}
		svc := newTestRankingService(repo, newTestConfig())
		entries, total, err := svc.GetRanking(ctx, model.RankingTypeCombatPower, &model.PageRequest{Page: 2, PageSize: 10})
		if err != nil {
			t.Fatal(err)
		}
		if total != 100 {
			t.Errorf("expected total 100, got %d", total)
		}
		if len(entries) != 10 {
			t.Errorf("expected 10 entries, got %d", len(entries))
		}
	})

	t.Run("page normalization applied", func(t *testing.T) {
		repo := &mockRankingRepo{
			getRankingByPageFn: func(_ context.Context, _ model.RankingType, page *model.PageRequest) ([]*model.RankingEntry, int64, error) {
				if page.Page != 1 {
					t.Errorf("expected page clamped to 1, got %d", page.Page)
				}
				if page.PageSize != 20 {
					t.Errorf("expected pageSize clamped to 20, got %d", page.PageSize)
				}
				return []*model.RankingEntry{}, 0, nil
			},
		}
		svc := newTestRankingService(repo, newTestConfig())
		_, _, err := svc.GetRanking(ctx, model.RankingTypeWealth, &model.PageRequest{Page: 0, PageSize: 0})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("invalid ranking type", func(t *testing.T) {
		svc := newTestRankingService(&mockRankingRepo{}, newTestConfig())
		_, _, err := svc.GetRanking(ctx, "invalid", &model.PageRequest{Page: 1, PageSize: 20})
		if err != ErrInvalidRankingType {
			t.Errorf("expected ErrInvalidRankingType, got %v", err)
		}
	})
}

// ============================================================================
// Ranking type validation
// ============================================================================

func TestRankingTypeValidationAllMethods(t *testing.T) {
	ctx := context.Background()
	validTypes := []model.RankingType{
		model.RankingTypeRealm,
		model.RankingTypeCombatPower,
		model.RankingTypeWealth,
		model.RankingTypeSect,
	}
	invalidTypes := []model.RankingType{"", "invalid", "REALM", " combat "}

	for _, rt := range validTypes {
		t.Run("valid type "+string(rt), func(t *testing.T) {
			svc := newTestRankingService(&mockRankingRepo{}, newTestConfig())
			// UpdateScore
			if err := svc.UpdateScore(ctx, rt, 1, 100, "a", "a"); err != nil {
				t.Errorf("UpdateScore: unexpected error: %v", err)
			}
		})
	}

	for _, rt := range invalidTypes {
		t.Run("invalid type "+string(rt), func(t *testing.T) {
			svc := newTestRankingService(&mockRankingRepo{}, newTestConfig())

			if err := svc.UpdateScore(ctx, rt, 1, 100, "a", "a"); err != ErrInvalidRankingType {
				t.Errorf("UpdateScore: expected ErrInvalidRankingType, got %v", err)
			}
			if _, err := svc.GetTopPlayers(ctx, rt, 10); err != ErrInvalidRankingType {
				t.Errorf("GetTopPlayers: expected ErrInvalidRankingType, got %v", err)
			}
			if _, _, err := svc.GetRanking(ctx, rt, &model.PageRequest{}); err != ErrInvalidRankingType {
				t.Errorf("GetRanking: expected ErrInvalidRankingType, got %v", err)
			}
			if _, _, _, err := svc.GetPlayerRank(ctx, rt, 1); err != ErrInvalidRankingType {
				t.Errorf("GetPlayerRank: expected ErrInvalidRankingType, got %v", err)
			}
			if err := svc.BatchUpdate(ctx, rt, []*model.RankingEntry{{PlayerID: 1}}); err != ErrInvalidRankingType {
				t.Errorf("BatchUpdate: expected ErrInvalidRankingType, got %v", err)
			}
		})
	}
}

// ============================================================================
// Score decay calculation
// ============================================================================

func TestApplyDecay(t *testing.T) {
	ctx := context.Background()

	t.Run("combat decay 3% per day after 7 days", func(t *testing.T) {
		var (
			appliedPlayers []uint64
			appliedRates   []float64
		)
		repo := &mockRankingRepo{
			getInactivePlayersFn: func(_ context.Context, rt model.RankingType, deadline time.Time) ([]uint64, error) {
				if rt == model.RankingTypeCombatPower {
					// Verify deadline is ~7 days ago
					expected := time.Now().AddDate(0, 0, -7)
					if deadline.Before(expected.Add(-time.Hour)) || deadline.After(expected.Add(time.Hour)) {
						t.Errorf("combat deadline should be ~7 days ago, got %v", deadline)
					}
					return []uint64{101, 102}, nil
				}
				return nil, nil
			},
			applyDecayToPlayerFn: func(_ context.Context, _ model.RankingType, playerID uint64, decayRate float64) error {
				appliedPlayers = append(appliedPlayers, playerID)
				appliedRates = append(appliedRates, decayRate)
				return nil
			},
		}
		svc := newTestRankingService(repo, newTestConfig())
		svc.applyDecay(ctx)

		if len(appliedPlayers) != 2 {
			t.Fatalf("expected 2 decay applications, got %d", len(appliedPlayers))
		}
		if appliedPlayers[0] != 101 {
			t.Errorf("first player = %d; want 101", appliedPlayers[0])
		}
		if appliedPlayers[1] != 102 {
			t.Errorf("second player = %d; want 102", appliedPlayers[1])
		}
		for i, rate := range appliedRates {
			if rate < 0.029 || rate > 0.031 {
				t.Errorf("rate[%d] = %v; want ~0.03", i, rate)
			}
		}
	})

	t.Run("wealth decay 2% per day after 14 days", func(t *testing.T) {
		var (
			appliedType     model.RankingType
			appliedPlayerID uint64
			appliedRate     float64
		)
		repo := &mockRankingRepo{
			getInactivePlayersFn: func(_ context.Context, rt model.RankingType, deadline time.Time) ([]uint64, error) {
				if rt == model.RankingTypeWealth {
					expected := time.Now().AddDate(0, 0, -14)
					if deadline.Before(expected.Add(-time.Hour)) || deadline.After(expected.Add(time.Hour)) {
						t.Errorf("wealth deadline should be ~14 days ago, got %v", deadline)
					}
					return []uint64{201}, nil
				}
				return nil, nil
			},
			applyDecayToPlayerFn: func(_ context.Context, rt model.RankingType, playerID uint64, decayRate float64) error {
				appliedType = rt
				appliedPlayerID = playerID
				appliedRate = decayRate
				return nil
			},
		}
		svc := newTestRankingService(repo, newTestConfig())
		svc.applyDecay(ctx)

		if appliedPlayerID != 201 {
			t.Errorf("expected player 201, got %d", appliedPlayerID)
		}
		if appliedRate < 0.019 || appliedRate > 0.021 {
			t.Errorf("expected decay rate ~0.02, got %v", appliedRate)
		}
		_ = appliedType
	})

	t.Run("realm and sect have no decay", func(t *testing.T) {
		callCount := 0
		repo := &mockRankingRepo{
			getInactivePlayersFn: func(_ context.Context, _ model.RankingType, _ time.Time) ([]uint64, error) {
				callCount++
				return nil, nil
			},
		}
		svc := newTestRankingService(repo, newTestConfig())
		svc.applyDecay(ctx)
		// Only combat and wealth have decay enabled
		if callCount != 2 {
			t.Errorf("expected 2 GetInactivePlayers calls (combat, wealth), got %d", callCount)
		}
	})

	t.Run("no inactive players is a no-op", func(t *testing.T) {
		applyCalled := false
		repo := &mockRankingRepo{
			getInactivePlayersFn: func(_ context.Context, _ model.RankingType, _ time.Time) ([]uint64, error) {
				return []uint64{}, nil
			},
			applyDecayToPlayerFn: func(_ context.Context, _ model.RankingType, _ uint64, _ float64) error {
				applyCalled = true
				return nil
			},
		}
		svc := newTestRankingService(repo, newTestConfig())
		svc.applyDecay(ctx)
		if applyCalled {
			t.Error("applyDecayToPlayer should not be called when no inactive players")
		}
	})

	t.Run("decay rate capped at 50%", func(t *testing.T) {
		repo := &mockRankingRepo{
			getInactivePlayersFn: func(_ context.Context, _ model.RankingType, _ time.Time) ([]uint64, error) {
				return []uint64{1}, nil
			},
			applyDecayToPlayerFn: func(_ context.Context, _ model.RankingType, _ uint64, decayRate float64) error {
				if decayRate > 0.5 {
					t.Errorf("decay rate should be capped at 0.5, got %v", decayRate)
				}
				return nil
			},
		}
		svc := newTestRankingService(repo, newTestConfig())
		svc.applyDecay(ctx)
	})
}

// ============================================================================
// Snapshot cache
// ============================================================================

func TestSnapshotCache(t *testing.T) {
	ctx := context.Background()

	makeEntries := func(n int) []*model.RankingEntry {
		entries := make([]*model.RankingEntry, n)
		for i := 0; i < n; i++ {
			entries[i] = &model.RankingEntry{PlayerID: uint64(i + 1), Score: float64(100 - i), Rank: int32(i + 1)}
		}
		return entries
	}

	t.Run("cache populated after first miss", func(t *testing.T) {
		callCount := 0
		repo := &mockRankingRepo{
			getTopNFn: func(_ context.Context, _ model.RankingType, _ int64) ([]*model.RankingEntry, error) {
				callCount++
				return makeEntries(10), nil
			},
		}
		svc := newTestRankingService(repo, newTestConfig())

		// Miss -> populate cache
		_, _ = svc.GetTopPlayers(ctx, model.RankingTypeRealm, 5)
		if callCount != 1 {
			t.Errorf("expected 1 call on miss, got %d", callCount)
		}

		// Hit -> use cache
		_, _ = svc.GetTopPlayers(ctx, model.RankingTypeRealm, 5)
		if callCount != 1 {
			t.Errorf("expected 0 additional calls on hit, got %d", callCount)
		}
	})

	t.Run("cache expires after refresh interval", func(t *testing.T) {
		callCount := 0
		repo := &mockRankingRepo{
			getTopNFn: func(_ context.Context, _ model.RankingType, _ int64) ([]*model.RankingEntry, error) {
				callCount++
				return makeEntries(10), nil
			},
		}
		cfg := newTestConfig()
		cfg.CacheRefreshInterval = 0 // make cache entries immediately expired
		svc := newTestRankingService(repo, cfg)

		_, _ = svc.GetTopPlayers(ctx, model.RankingTypeRealm, 5)
		_, _ = svc.GetTopPlayers(ctx, model.RankingTypeRealm, 5)
		if callCount != 2 {
			t.Errorf("expected 2 calls (cache expired), got %d", callCount)
		}
	})

	t.Run("empty result does not cache", func(t *testing.T) {
		callCount := 0
		repo := &mockRankingRepo{
			getTopNFn: func(_ context.Context, _ model.RankingType, _ int64) ([]*model.RankingEntry, error) {
				callCount++
				return []*model.RankingEntry{}, nil
			},
		}
		svc := newTestRankingService(repo, newTestConfig())

		_, _ = svc.GetTopPlayers(ctx, model.RankingTypeRealm, 5)
		_, _ = svc.GetTopPlayers(ctx, model.RankingTypeRealm, 5)
		// Both should miss because empty results don't get cached
		if callCount != 2 {
			t.Errorf("expected 2 calls (empty not cached), got %d", callCount)
		}
	})
}

// ============================================================================
// Service Stop
// ============================================================================

func TestServiceStop(t *testing.T) {
	svc := newTestRankingService(&mockRankingRepo{}, newTestConfig())
	// Stop should not panic
	svc.Stop()
	// Double stop should not panic
	svc.Stop()
}
