package state

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupTestStore(t *testing.T) (*Store, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() failed: %v", err)
	}
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })

	store := NewStore(client)
	return store, mr
}

// ---------- Set / Get roundtrip ----------

func TestSetAndGetRoundtrip(t *testing.T) {
	store, _ := setupTestStore(t)
	ctx := context.Background()

	t.Run("string value", func(t *testing.T) {
		err := store.Set(ctx, "greeting", "hello world")
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		var result string
		err = store.Get(ctx, "greeting", &result)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if result != "hello world" {
			t.Errorf("got %q, want %q", result, "hello world")
		}
	})

	t.Run("integer value", func(t *testing.T) {
		err := store.Set(ctx, "counter", 42)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		var result int
		err = store.Get(ctx, "counter", &result)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if result != 42 {
			t.Errorf("got %d, want %d", result, 42)
		}
	})

	t.Run("struct value", func(t *testing.T) {
		type Player struct {
			Name  string `json:"name"`
			Level int    `json:"level"`
		}
		player := Player{Name: "alice", Level: 10}

		err := store.Set(ctx, "player:1001", player)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		var result Player
		err = store.Get(ctx, "player:1001", &result)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if result.Name != "alice" || result.Level != 10 {
			t.Errorf("got %+v, want %+v", result, player)
		}
	})

	t.Run("overwrite existing key", func(t *testing.T) {
		store.Set(ctx, "key", "first")
		store.Set(ctx, "key", "second")

		var result string
		err := store.Get(ctx, "key", &result)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if result != "second" {
			t.Errorf("got %q, want %q", result, "second")
		}
	})
}

// ---------- Get returns error for non-existent key ----------

func TestGetNonExistentKey(t *testing.T) {
	store, _ := setupTestStore(t)
	ctx := context.Background()

	t.Run("missing key", func(t *testing.T) {
		var result string
		err := store.Get(ctx, "does-not-exist", &result)
		if err == nil {
			t.Fatal("expected error for non-existent key")
		}
		if err.Error() != "state: key does-not-exist not found" {
			t.Errorf("error = %q, want %q", err.Error(), "state: key does-not-exist not found")
		}
	})

	t.Run("missing key after flush and cache invalidation", func(t *testing.T) {
		// Set and flush a key, then delete it from the store
		store.Set(ctx, "temp", "value")
		store.Flush(ctx)

		// Manually remove from cache to simulate fresh state
		store.mu.Lock()
		delete(store.cache, "temp")
		delete(store.dirty, "temp")
		store.mu.Unlock()

		// Now delete from Redis directly (simulating external deletion)
		store.client.Del(ctx, "state:temp")

		var result string
		err := store.Get(ctx, "temp", &result)
		if err == nil {
			t.Fatal("expected error after deletion")
		}
	})
}

// ---------- Delete ----------

func TestDeleteRemovesKey(t *testing.T) {
	store, mr := setupTestStore(t)
	ctx := context.Background()

	store.Set(ctx, "delete-me", "to-be-deleted")

	// Key should be in cache and dirty
	store.mu.RLock()
	_, inCache := store.cache["delete-me"]
	_, inDirty := store.dirty["delete-me"]
	store.mu.RUnlock()

	if !inCache {
		t.Error("key should be in cache after Set")
	}
	if !inDirty {
		t.Error("key should be dirty after Set")
	}

	err := store.Delete(ctx, "delete-me")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Key should be removed from cache and dirty
	store.mu.RLock()
	_, inCache = store.cache["delete-me"]
	_, inDirty = store.dirty["delete-me"]
	store.mu.RUnlock()

	if inCache {
		t.Error("key should not be in cache after Delete")
	}
	if inDirty {
		t.Error("key should not be dirty after Delete")
	}

	// Redis key should be gone
	if mr.Exists("state:delete-me") {
		t.Error("key should not exist in Redis after Delete")
	}

	// Get should fail
	var result string
	err = store.Get(ctx, "delete-me", &result)
	if err == nil {
		t.Error("expected error getting deleted key")
	}
}

func TestDeleteNonExistentKey(t *testing.T) {
	store, _ := setupTestStore(t)
	ctx := context.Background()

	// Deleting a non-existent key should not error (Redis Del is idempotent)
	err := store.Delete(ctx, "never-existed")
	if err != nil {
		t.Errorf("Delete of non-existent key should not error, got: %v", err)
	}
}

// ---------- SetImmediate ----------

func TestSetImmediateWritesDirectly(t *testing.T) {
	store, mr := setupTestStore(t)
	ctx := context.Background()

	t.Run("writes directly to Redis", func(t *testing.T) {
		type Data struct {
			Value string `json:"value"`
		}
		data := Data{Value: "immediate"}

		err := store.SetImmediate(ctx, "direct-key", data)
		if err != nil {
			t.Fatalf("SetImmediate failed: %v", err)
		}

		// Should NOT be in cache
		store.mu.RLock()
		_, inCache := store.cache["direct-key"]
		store.mu.RUnlock()
		if inCache {
			t.Error("SetImmediate should not add to cache")
		}

		// Should be in Redis
		if !mr.Exists("state:direct-key") {
			t.Error("key should exist in Redis after SetImmediate")
		}
	})

	t.Run("retrievable via Get", func(t *testing.T) {
		store.SetImmediate(ctx, "get-test", "stored-immediately")

		var result string
		err := store.Get(ctx, "get-test", &result)
		if err != nil {
			t.Fatalf("Get after SetImmediate failed: %v", err)
		}
		if result != "stored-immediately" {
			t.Errorf("got %q, want %q", result, "stored-immediately")
		}
	})

	t.Run("marshal error", func(t *testing.T) {
		// Unmarshallable type (channel)
		err := store.SetImmediate(ctx, "bad-key", make(chan int))
		if err == nil {
			t.Error("expected error for unmarshalable type")
		}
	})
}

// ---------- Flush ----------

func TestFlushBatchesDirtyData(t *testing.T) {
	store, mr := setupTestStore(t)
	ctx := context.Background()

	store.Set(ctx, "flush-key-1", "value1")
	store.Set(ctx, "flush-key-2", "value2")
	store.Set(ctx, "flush-key-3", "value3")

	// All should be dirty
	store.mu.RLock()
	dirtyCount := len(store.dirty)
	store.mu.RUnlock()
	if dirtyCount != 3 {
		t.Errorf("expected 3 dirty keys, got %d", dirtyCount)
	}

	// Data should NOT be in Redis yet
	if mr.Exists("state:flush-key-1") {
		t.Error("data should not be in Redis before Flush")
	}

	err := store.Flush(ctx)
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Dirty should be cleared
	store.mu.RLock()
	dirtyCount = len(store.dirty)
	store.mu.RUnlock()
	if dirtyCount != 0 {
		t.Errorf("expected 0 dirty keys after Flush, got %d", dirtyCount)
	}

	// Data should be in Redis
	if !mr.Exists("state:flush-key-1") {
		t.Error("key-1 should be in Redis after Flush")
	}
	if !mr.Exists("state:flush-key-2") {
		t.Error("key-2 should be in Redis after Flush")
	}
	if !mr.Exists("state:flush-key-3") {
		t.Error("key-3 should be in Redis after Flush")
	}
}

func TestFlushNoDirtyKeys(t *testing.T) {
	store, _ := setupTestStore(t)
	ctx := context.Background()

	// Flush with no dirty keys should be a no-op
	err := store.Flush(ctx)
	if err != nil {
		t.Errorf("Flush with no dirty keys should succeed: %v", err)
	}
}

func TestFlushOnlyDirtyKeys(t *testing.T) {
	store, mr := setupTestStore(t)
	ctx := context.Background()

	// Set several keys so they're all dirty
	store.Set(ctx, "a", "alpha")
	store.Set(ctx, "b", "beta")
	store.Flush(ctx)

	// Now only "c" is dirty
	store.Set(ctx, "c", "charlie")

	// Manually mark "a" and "b" as not dirty (simulate what Flush did)
	store.mu.Lock()
	delete(store.dirty, "a")
	delete(store.dirty, "b")
	store.mu.Unlock()

	// Verify only "c" is dirty
	store.mu.RLock()
	dirtyCount := len(store.dirty)
	store.mu.RUnlock()
	if dirtyCount != 1 {
		t.Errorf("expected 1 dirty key, got %d", dirtyCount)
	}

	// Flush should only set "c" in Redis (others already there)
	err := store.Flush(ctx)
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// All keys should be in Redis
	if !mr.Exists("state:a") {
		t.Error("key 'a' should exist in Redis")
	}
	if !mr.Exists("state:b") {
		t.Error("key 'b' should exist in Redis")
	}
	if !mr.Exists("state:c") {
		t.Error("key 'c' should exist in Redis")
	}
}

func TestFlushEmptyAfterDelete(t *testing.T) {
	store, _ := setupTestStore(t)
	ctx := context.Background()

	store.Set(ctx, "temp", "value")
	store.Delete(ctx, "temp")

	// Dirty should be empty
	store.mu.RLock()
	dirtyCount := len(store.dirty)
	store.mu.RUnlock()
	if dirtyCount != 0 {
		t.Errorf("expected 0 dirty keys after delete, got %d", dirtyCount)
	}
}

// ---------- Auto-save ----------

func TestAutoSaveStartStop(t *testing.T) {
	store, _ := setupTestStore(t)
	ctx := context.Background()

	store.StartAutoSave(ctx)

	store.mu.RLock()
	running := store.running
	store.mu.RUnlock()
	if !running {
		t.Error("store should be running after StartAutoSave")
	}

	// Second StartAutoSave should be a no-op (not create a second goroutine)
	store.StartAutoSave(ctx)

	store.Stop()

	store.mu.RLock()
	running = store.running
	store.mu.RUnlock()
	if running {
		t.Error("store should not be running after Stop")
	}

	// Second Stop should be safe (no panic)
	store.Stop()
}

func TestAutoSaveFlushesOnStop(t *testing.T) {
	store, mr := setupTestStore(t)
	ctx := context.Background()

	store.StartAutoSave(ctx)
	store.Set(ctx, "auto-flush-key", "auto-value")

	// Stop triggers a final flush
	store.Stop()

	// Give the goroutine a moment to execute the final flush
	time.Sleep(50 * time.Millisecond)

	// Data should be in Redis
	if !mr.Exists("state:auto-flush-key") {
		t.Error("data should be flushed to Redis on Stop")
	}

	// Should be retrievable via Get
	var result string
	err := store.Get(ctx, "auto-flush-key", &result)
	if err != nil {
		t.Fatalf("Get after auto-save stop failed: %v", err)
	}
	if result != "auto-value" {
		t.Errorf("got %q, want %q", result, "auto-value")
	}
}

func TestAutoSaveContextCancellation(t *testing.T) {
	store, _ := setupTestStore(t)

	ctx, cancel := context.WithCancel(context.Background())
	store.StartAutoSave(ctx)

	// Cancel context - the goroutine should exit without flushing
	cancel()

	// Wait for goroutine to exit
	time.Sleep(50 * time.Millisecond)

	// Store should still be running (Until Stop() is called, the goroutine exits due to ctx.Done())
	store.mu.RLock()
	running := store.running
	store.mu.RUnlock()
	if !running {
		t.Error("store should still be considered running after ctx cancel")
	}

	// Clean up
	store.Stop()
}

// ---------- Batch size threshold ----------

func TestSetTriggersFlushAtBatchThreshold(t *testing.T) {
	store, mr := setupTestStore(t)
	ctx := context.Background()

	// Reduce batch size for testing
	store.batch = 5

	// Set 4 keys - no flush expected
	for i := 0; i < 4; i++ {
		err := store.Set(ctx, "batch-key-"+string(rune('a'+i)), i)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	// Verify dirty count is 4
	store.mu.RLock()
	dirtyCount := len(store.dirty)
	store.mu.RUnlock()
	if dirtyCount != 4 {
		t.Errorf("expected 4 dirty keys, got %d", dirtyCount)
	}

	// Verify nothing in Redis yet
	if mr.Exists("state:batch-key-a") {
		t.Error("data should not be in Redis before batch threshold")
	}

	// 5th Set should trigger flush
	err := store.Set(ctx, "batch-key-e", 5)
	if err != nil {
		t.Fatalf("5th Set failed: %v", err)
	}

	// After flush, dirty should be empty
	store.mu.RLock()
	dirtyCount = len(store.dirty)
	store.mu.RUnlock()
	if dirtyCount != 0 {
		t.Errorf("expected 0 dirty keys after flush, got %d", dirtyCount)
	}

	// Data should be in Redis
	if !mr.Exists("state:batch-key-a") {
		t.Error("data should be in Redis after batch threshold flush")
	}
}

// ---------- Concurrent safety ----------

func TestConcurrentSetSafety(t *testing.T) {
	store, _ := setupTestStore(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := "concurrent-key"
			// Use the same key to test concurrent writes to the same map entry
			err := store.Set(ctx, key, idx)
			if err != nil {
				t.Errorf("concurrent Set failed: %v", err)
			}
		}(i)
	}
	wg.Wait()

	// The key should still be in cache (last write wins)
	store.mu.RLock()
	val, inCache := store.cache["concurrent-key"]
	_, inDirty := store.dirty["concurrent-key"]
	store.mu.RUnlock()

	if !inCache {
		t.Error("concurrent-key should be in cache after concurrent Sets")
	}
	if !inDirty {
		t.Error("concurrent-key should be dirty after concurrent Sets")
	}
	if val == nil {
		t.Error("concurrent-key should have a non-nil value")
	}
}

func TestConcurrentSetDifferentKeysSafety(t *testing.T) {
	store, _ := setupTestStore(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	numKeys := 100

	for i := 0; i < numKeys; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := "key-" + string(rune('0'+idx%10))
			err := store.Set(ctx, key, idx)
			if err != nil {
				t.Errorf("Set failed: %v", err)
			}
		}(i)
	}
	wg.Wait()

	store.mu.RLock()
	cacheLen := len(store.cache)
	store.mu.RUnlock()

	if cacheLen == 0 {
		t.Error("cache should have entries after concurrent Sets")
	}
}

func TestConcurrentGetAndSetSafety(t *testing.T) {
	store, _ := setupTestStore(t)
	ctx := context.Background()

	store.Set(ctx, "shared", "initial")

	var wg sync.WaitGroup

	// Concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var val string
			_ = store.Get(ctx, "shared", &val)
		}()
	}

	// Concurrent writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			store.Set(ctx, "shared", idx)
		}(i)
	}

	wg.Wait()
}

// ---------- copyValue helper ----------

func TestCopyValue(t *testing.T) {
	t.Run("struct to struct pointer", func(t *testing.T) {
		type Data struct {
			Name string
			Age  int
		}
		src := Data{Name: "test", Age: 25}
		var dst Data

		err := copyValue(src, &dst)
		if err != nil {
			t.Fatalf("copyValue failed: %v", err)
		}
		if dst.Name != "test" || dst.Age != 25 {
			t.Errorf("got %+v, want %+v", dst, src)
		}
	})

	t.Run("string to string pointer", func(t *testing.T) {
		var dst string
		err := copyValue("hello", &dst)
		if err != nil {
			t.Fatalf("copyValue failed: %v", err)
		}
		if dst != "hello" {
			t.Errorf("got %q, want %q", dst, "hello")
		}
	})

	t.Run("int to int pointer", func(t *testing.T) {
		var dst int
		err := copyValue(42, &dst)
		if err != nil {
			t.Fatalf("copyValue failed: %v", err)
		}
		if dst != 42 {
			t.Errorf("got %d, want %d", dst, 42)
		}
	})

	t.Run("map to map pointer", func(t *testing.T) {
		src := map[string]int{"a": 1, "b": 2}
		dst := make(map[string]int)

		err := copyValue(src, &dst)
		if err != nil {
			t.Fatalf("copyValue failed: %v", err)
		}
		if dst["a"] != 1 || dst["b"] != 2 {
			t.Errorf("got %v, want %v", dst, src)
		}
	})
}
