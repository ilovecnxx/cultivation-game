package client

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"RequestTimeout", cfg.RequestTimeout, 5 * time.Second},
		{"RetryMaxWait", cfg.RetryMaxWait, 30 * time.Second},
		{"MaxRetries", cfg.MaxRetries, 3},
		{"RetryBackoffMin", cfg.RetryBackoffMin, 100 * time.Millisecond},
		{"RetryBackoffMax", cfg.RetryBackoffMax, 5 * time.Second},
		{"CircuitThreshold", cfg.CircuitThreshold, 5},
		{"CircuitTimeout", cfg.CircuitTimeout, 30 * time.Second},
		{"HalfOpenMaxReqs", cfg.HalfOpenMaxReqs, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestNewWithNilConfig(t *testing.T) {
	sc := New(nil)
	if sc == nil {
		t.Fatal("New(nil) returned nil")
	}

	// Should use defaults
	if sc.config.RequestTimeout != 5*time.Second {
		t.Errorf("expected default RequestTimeout, got %v", sc.config.RequestTimeout)
	}
	if sc.breakers == nil {
		t.Error("breakers map should be initialized")
	}
}

func TestCircuitStateString(t *testing.T) {
	tests := []struct {
		state CircuitState
		want  string
	}{
		{CircuitClosed, "closed"},
		{CircuitHalfOpen, "half-open"},
		{CircuitOpen, "open"},
		{CircuitState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCircuitBreakerInitialState(t *testing.T) {
	cb := newCircuitBreaker(5, 30*time.Second, 3)

	if cb.state != CircuitClosed {
		t.Errorf("initial state = %v, want %v", cb.state, CircuitClosed)
	}
	if cb.failureCount != 0 {
		t.Errorf("initial failureCount = %d, want 0", cb.failureCount)
	}
	if cb.threshold != 5 {
		t.Errorf("threshold = %d, want 5", cb.threshold)
	}
	if cb.timeout != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", cb.timeout)
	}
	if cb.halfOpenMax != 3 {
		t.Errorf("halfOpenMax = %d, want 3", cb.halfOpenMax)
	}
}

func TestCircuitBreakerClosedStateAllows(t *testing.T) {
	cb := newCircuitBreaker(5, 30*time.Second, 3)

	// In closed state, allow() should always return true
	for i := 0; i < 10; i++ {
		if !cb.allow() {
			t.Errorf("allow() returned false in closed state on iteration %d", i)
		}
	}
}

func TestCircuitBreakerOpenStateDenies(t *testing.T) {
	cb := newCircuitBreaker(5, 30*time.Second, 3)
	cb.state = CircuitOpen
	cb.lastFailTime = time.Now()

	// In open state (before timeout), allow() should deny
	if cb.allow() {
		t.Error("allow() returned true in open state")
	}
}

func TestCircuitBreakerOpenToHalfOpenAfterTimeout(t *testing.T) {
	cb := newCircuitBreaker(5, 50*time.Millisecond, 3)
	cb.state = CircuitOpen
	cb.lastFailTime = time.Now().Add(-100 * time.Millisecond)

	// After timeout has elapsed, allow() should transition to half-open and allow
	if !cb.allow() {
		t.Fatal("allow() should return true when transitioning from open to half-open")
	}
	if cb.state != CircuitHalfOpen {
		t.Errorf("state = %v, want %v", cb.state, CircuitHalfOpen)
	}
	if cb.halfOpenReqs != 0 {
		t.Errorf("halfOpenReqs = %d, want 0 after transition", cb.halfOpenReqs)
	}
}

func TestCircuitBreakerHalfOpenWithinTimeout(t *testing.T) {
	cb := newCircuitBreaker(5, 30*time.Second, 3)
	cb.state = CircuitOpen
	cb.lastFailTime = time.Now().Add(-1 * time.Second) // Not enough time

	// Should still be open, no transition
	if cb.allow() {
		t.Error("allow() should return false before timeout elapses")
	}
	if cb.state != CircuitOpen {
		t.Errorf("state = %v, want %v (should remain open)", cb.state, CircuitOpen)
	}
}

func TestCircuitBreakerHalfOpenAllowsLimitedRequests(t *testing.T) {
	halfOpenMax := 2
	cb := newCircuitBreaker(5, 30*time.Second, halfOpenMax)
	cb.state = CircuitHalfOpen

	// First request should be allowed
	if !cb.allow() {
		t.Error("first half-open request should be allowed")
	}

	// Second should be allowed
	if !cb.allow() {
		t.Error("second half-open request should be allowed")
	}

	// Third should be denied (exceeds halfOpenMax)
	if cb.allow() {
		t.Error("third half-open request should be denied (exceeds limit)")
	}

	// halfOpenReqs should match the number of allowed requests
	if cb.halfOpenReqs != halfOpenMax {
		t.Errorf("halfOpenReqs = %d, want %d", cb.halfOpenReqs, halfOpenMax)
	}
}

func TestCircuitBreakerSuccessResetsFailureCount(t *testing.T) {
	cb := newCircuitBreaker(5, 30*time.Second, 3)
	cb.failureCount = 3
	cb.state = CircuitHalfOpen
	cb.halfOpenReqs = 1

	cb.recordSuccess()

	if cb.failureCount != 0 {
		t.Errorf("failureCount = %d, want 0 after success", cb.failureCount)
	}
	if cb.state != CircuitClosed {
		t.Errorf("state = %v, want %v after success in half-open", cb.state, CircuitClosed)
	}
	if cb.halfOpenReqs != 0 {
		t.Errorf("halfOpenReqs = %d, want 0 after transition to closed", cb.halfOpenReqs)
	}
}

func TestCircuitBreakerFailureIncrementsCount(t *testing.T) {
	cb := newCircuitBreaker(5, 30*time.Second, 3)

	// Record failures sequentially
	for i := 1; i <= 4; i++ {
		cb.recordFailure()
		if cb.failureCount != i {
			t.Errorf("after %d failures: failureCount = %d, want %d", i, cb.failureCount, i)
		}
		if cb.state != CircuitClosed {
			t.Errorf("state should remain closed after %d failures, got %v", i, cb.state)
		}
	}
}

func TestCircuitBreakerThresholdTriggersOpen(t *testing.T) {
	cb := newCircuitBreaker(5, 30*time.Second, 3)

	// Hit the threshold exactly
	for i := 0; i < 5; i++ {
		cb.recordFailure()
	}

	if cb.state != CircuitOpen {
		t.Errorf("state = %v, want %v after threshold reached", cb.state, CircuitOpen)
	}
	if cb.failureCount != 5 {
		t.Errorf("failureCount = %d, want 5", cb.failureCount)
	}
	if cb.lastFailTime.IsZero() {
		t.Error("lastFailTime should be set")
	}
}

func TestCircuitBreakerHalfOpenFailureReturnsToOpen(t *testing.T) {
	cb := newCircuitBreaker(5, 30*time.Second, 3)
	cb.state = CircuitHalfOpen

	cb.recordFailure()

	if cb.state != CircuitOpen {
		t.Errorf("state = %v, want %v after half-open failure", cb.state, CircuitOpen)
	}
}

func TestCircuitBreakerSuccessInClosedStaysClosed(t *testing.T) {
	cb := newCircuitBreaker(5, 30*time.Second, 3)
	cb.failureCount = 2

	cb.recordSuccess()

	if cb.state != CircuitClosed {
		t.Errorf("state should remain closed, got %v", cb.state)
	}
	if cb.failureCount != 0 {
		t.Errorf("failureCount = %d, want 0", cb.failureCount)
	}
}

func TestGetBreakerCreatesAndCaches(t *testing.T) {
	sc := New(DefaultConfig())

	// First call should create a new breaker
	cb1 := sc.getBreaker("player")
	if cb1 == nil {
		t.Fatal("getBreaker returned nil")
	}
	if cb1.state != CircuitClosed {
		t.Errorf("new breaker state = %v, want %v", cb1.state, CircuitClosed)
	}
	if cb1.threshold != sc.config.CircuitThreshold {
		t.Errorf("threshold = %d, want %d", cb1.threshold, sc.config.CircuitThreshold)
	}
	if cb1.timeout != sc.config.CircuitTimeout {
		t.Errorf("timeout = %v, want %v", cb1.timeout, sc.config.CircuitTimeout)
	}

	// Second call with same service name should return the same breaker
	cb2 := sc.getBreaker("player")
	if cb1 != cb2 {
		t.Error("getBreaker should return the same instance for the same service")
	}

	// Different service should get a different breaker
	cb3 := sc.getBreaker("world")
	if cb1 == cb3 {
		t.Error("different services should have different breakers")
	}
}

func TestGetBreakerConcurrentAccess(t *testing.T) {
	sc := New(DefaultConfig())

	// Concurrently access getBreaker for the same service
	done := make(chan struct{})
	for i := 0; i < 20; i++ {
		go func() {
			cb := sc.getBreaker("concurrent-service")
			if cb == nil {
				t.Error("getBreaker returned nil")
			}
			done <- struct{}{}
		}()
	}

	for i := 0; i < 20; i++ {
		<-done
	}

	// Should have exactly one breaker for that service
	sc.mu.RLock()
	count := len(sc.breakers)
	sc.mu.RUnlock()

	if count != 1 {
		t.Errorf("expected 1 breaker for concurrent-service, got %d total breakers", count)
	}
}

func TestConfigCircuitThresholdZero(t *testing.T) {
	// When threshold is 0, the first failure should trigger open state
	cfg := DefaultConfig()
	cfg.CircuitThreshold = 0
	sc := New(cfg)

	cb := sc.getBreaker("test")
	cb.recordFailure()

	if cb.state != CircuitOpen {
		t.Errorf("with threshold=0, first failure should open circuit, got state %v", cb.state)
	}
}

func TestNewWithCustomConfig(t *testing.T) {
	cfg := &Config{
		RequestTimeout:   10 * time.Second,
		RetryMaxWait:     60 * time.Second,
		MaxRetries:       5,
		RetryBackoffMin:  200 * time.Millisecond,
		RetryBackoffMax:  10 * time.Second,
		CircuitThreshold: 10,
		CircuitTimeout:   60 * time.Second,
		HalfOpenMaxReqs:  5,
	}

	sc := New(cfg)
	if sc.config.RequestTimeout != 10*time.Second {
		t.Errorf("RequestTimeout = %v, want 10s", sc.config.RequestTimeout)
	}
	if sc.config.CircuitThreshold != 10 {
		t.Errorf("CircuitThreshold = %d, want 10", sc.config.CircuitThreshold)
	}
}

func TestGetStats(t *testing.T) {
	sc := New(DefaultConfig())
	sc.getBreaker("svc-a")
	sc.getBreaker("svc-b")

	stats := sc.GetStats()
	if len(stats) != 2 {
		t.Errorf("expected 2 services in stats, got %d", len(stats))
	}

	for svc, stat := range stats {
		if stat == "" {
			t.Errorf("empty stat for service %q", svc)
		}
	}
}
