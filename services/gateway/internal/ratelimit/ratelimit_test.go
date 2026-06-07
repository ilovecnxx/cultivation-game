// Package ratelimit 令牌桶限流器单元测试。
package ratelimit

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// TokenBucket — Allow / AllowN
// ---------------------------------------------------------------------------

func TestTokenBucket_Allow(t *testing.T) {
	tb := NewTokenBucket(10, 5)
	if !tb.Allow() {
		t.Error("expected Allow to return true when bucket is full")
	}
}

func TestTokenBucket_DenyWhenEmpty(t *testing.T) {
	tb := NewTokenBucket(10, 1)

	// Drain the bucket.
	if !tb.Allow() {
		t.Fatal("expected Allow on initial token")
	}

	// Second call with rate=10, capacity=1: tokens = 1, after first call 0.
	// With elapsed ~0ms, refill adds ~0 tokens, so still 0 < 1.
	if tb.Allow() {
		t.Error("expected Allow to return false when bucket is empty and no time has passed")
	}
}

func TestTokenBucket_AllowN(t *testing.T) {
	t.Run("consume within capacity", func(t *testing.T) {
		tb := NewTokenBucket(10, 5)
		if !tb.AllowN(5) {
			t.Error("AllowN(5) should succeed for full bucket")
		}
		// Empty bucket; no time has passed, so no refill.
		if tb.AllowN(1) {
			t.Error("AllowN(1) should fail for empty bucket")
		}
	})

	t.Run("partial consumption", func(t *testing.T) {
		tb := NewTokenBucket(10, 5)
		if !tb.AllowN(3) {
			t.Error("AllowN(3) should succeed for full bucket")
		}
		if !tb.AllowN(2) {
			t.Error("AllowN(2) should succeed with 2 remaining")
		}
	})

	t.Run("consume exactly capacity", func(t *testing.T) {
		tb := NewTokenBucket(10, 5)
		if !tb.AllowN(5) {
			t.Error("AllowN(5) should succeed for full bucket")
		}
	})

	t.Run("over capacity", func(t *testing.T) {
		tb := NewTokenBucket(10, 3)
		if tb.AllowN(10) {
			t.Error("AllowN(10) should fail when bucket only has 3")
		}
	})
}

func TestTokenBucket_AllowN_Zero(t *testing.T) {
	tb := NewTokenBucket(10, 5)
	// Consuming 0 tokens should always succeed.
	if !tb.AllowN(0) {
		t.Error("AllowN(0) should always return true")
	}

	// After draining, consuming 0 should still succeed.
	tb.AllowN(5) // drain
	if !tb.AllowN(0) {
		t.Error("AllowN(0) should return true even when bucket is empty")
	}
}

func TestTokenBucket_AllowN_Negative(t *testing.T) {
	tb := NewTokenBucket(10, 5)
	// NOTE: AllowN(-1) currently returns true because the comparison
	// `tb.tokens >= float64(-1)` always holds, and the subtraction
	// `tb.tokens -= float64(-1)` effectively adds a token.
	// This documents the existing behaviour.
	preCount := tb.tokens
	result := tb.AllowN(-1)
	if !result {
		t.Error("AllowN(-1) returned false (current code allows negative n)")
	}
	// Tokens increased (or stayed at capacity).
	if tb.tokens < preCount {
		t.Error("AllowN(-1) should not decrease the token count")
	}
}

// ---------------------------------------------------------------------------
// TokenBucket — refill over time
// ---------------------------------------------------------------------------

func TestTokenBucket_RefillOverTime(t *testing.T) {
	// High refill rate so we can test refill with a short wait.
	tb := NewTokenBucket(1000, 1)

	// Drain the bucket.
	tb.Allow() // tokens -> 0

	// Wait 10ms => ~10 tokens should have been added (rate 1000/s),
	// but capped at capacity 1.
	time.Sleep(10 * time.Millisecond)

	if !tb.Allow() {
		t.Error("expected Allow to return true after refill")
	}
}

func TestTokenBucket_Available(t *testing.T) {
	tb := NewTokenBucket(10, 5)

	// Initially: capacity = 5 tokens.
	avail := tb.Available()
	if avail < 4.5 || avail > 5.5 {
		t.Errorf("Available() = %f, want ~5", avail)
	}

	// Consume 2 tokens.
	tb.AllowN(2)
	avail = tb.Available()
	// Should show ~3 tokens (with tiny refill from elapsed time).
	if avail < 2.5 || avail > 3.5 {
		t.Errorf("Available() after consuming 2 = %f, want ~3", avail)
	}
}

// ---------------------------------------------------------------------------
// TokenBucket — edge cases
// ---------------------------------------------------------------------------

func TestTokenBucket_ZeroRate(t *testing.T) {
	tb := NewTokenBucket(0, 5)

	// Can use initial tokens but never refill.
	tb.AllowN(5) // drain all tokens

	// No refill (rate=0).
	time.Sleep(10 * time.Millisecond)

	if tb.Allow() {
		t.Error("expected Allow to return false when rate is 0 and bucket is drained")
	}
}

func TestTokenBucket_BurstCapacity(t *testing.T) {
	// Low rate but high burst capacity.
	tb := NewTokenBucket(1, 100)

	// Should be able to consume all 100 tokens at once (burst).
	if !tb.AllowN(100) {
		t.Error("expected AllowN(100) on full bucket with capacity 100")
	}

	// After draining, only refill happens (1 token/second).
	time.Sleep(50 * time.Millisecond)
	if tb.AllowN(10) {
		t.Error("expected AllowN(10) to fail after only 50ms refill at rate=1")
	}
}

func TestTokenBucket_ZeroCapacity(t *testing.T) {
	tb := NewTokenBucket(10, 0)
	if tb.Allow() {
		t.Error("expected Allow to return false for zero-capacity bucket")
	}
	if tb.AllowN(1) {
		t.Error("expected AllowN to return false for zero-capacity bucket")
	}
}

// ---------------------------------------------------------------------------
// RateLimiter — multi-key
// ---------------------------------------------------------------------------

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(10, 5)

	if !rl.Allow("player_1") {
		t.Error("expected Allow to return true for new key")
	}
}

func TestRateLimiter_MultipleKeys(t *testing.T) {
	rl := NewRateLimiter(10, 1)

	if !rl.Allow("player_a") {
		t.Fatal("expected Allow for player_a")
	}
	if !rl.Allow("player_b") {
		t.Fatal("expected Allow for player_b")
	}
	// Both buckets are now empty (0 tokens each). Allow should return false
	// for both unless refill has happened.
	if rl.Allow("player_a") {
		t.Error("expected Allow for player_a to be false after draining")
	}
	if rl.Allow("player_b") {
		t.Error("expected Allow for player_b to be false after draining")
	}
}

func TestRateLimiter_AllowN(t *testing.T) {
	rl := NewRateLimiter(10, 5)

	allowed := rl.AllowN("player_1", 3)
	if !allowed {
		t.Fatal("AllowN(3) should succeed for full bucket")
	}

	// Only 2 tokens left.
	allowed = rl.AllowN("player_1", 2)
	if !allowed {
		t.Fatal("AllowN(2) should succeed with 2 remaining")
	}

	// Now empty.
	allowed = rl.AllowN("player_1", 1)
	if allowed {
		t.Error("AllowN(1) should fail for empty bucket")
	}
}

func TestRateLimiter_GetBucket(t *testing.T) {
	rl := NewRateLimiter(10, 5)

	b1 := rl.GetBucket("key_a")
	b2 := rl.GetBucket("key_a")
	b3 := rl.GetBucket("key_b")

	if b1 != b2 {
		t.Error("GetBucket should return the same bucket for the same key")
	}
	if b1 == b3 {
		t.Error("GetBucket should return different buckets for different keys")
	}
}

func TestRateLimiter_RemoveBucket(t *testing.T) {
	rl := NewRateLimiter(10, 5)

	// Create a bucket by accessing it.
	b1 := rl.GetBucket("key_remove")
	rl.RemoveBucket("key_remove")

	// After removal, requesting the same key should create a new bucket.
	b2 := rl.GetBucket("key_remove")
	if b1 == b2 {
		t.Error("RemoveBucket should delete the old bucket; GetBucket should create a new one")
	}
}

// ---------------------------------------------------------------------------
// Concurrent access safety
// ---------------------------------------------------------------------------

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := NewRateLimiter(1000, 100)
	var wg sync.WaitGroup

	// 20 goroutines each performing 100 operations on 5 keys.
	const goroutines = 20
	const opsPerGoroutine = 100
	const numKeys = 5

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("player_%d", id%numKeys)
			for j := 0; j < opsPerGoroutine; j++ {
				rl.Allow(key)
				rl.AllowN(key, 2)
				rl.GetBucket(key)
			}
		}(i)
	}
	wg.Wait()
	// If we get here without data race, concurrent access is safe.
	// Run with "go test -race" to verify.
}

func TestTokenBucket_ConcurrentAccess(t *testing.T) {
	tb := NewTokenBucket(1000, 100)
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				tb.Allow()
				tb.AllowN(3)
				tb.Available()
			}
		}()
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// Large-scale stress
// ---------------------------------------------------------------------------

func TestRateLimiter_Stress(t *testing.T) {
	rl := NewRateLimiter(1000, 100)

	// Create 1000 different keys and perform operations on them.
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("stress_%d", i)
		rl.GetBucket(key)
	}
	if !rl.Allow("stress_0") {
		t.Error("expected Allow to succeed for stress key")
	}
}
