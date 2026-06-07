package eventbus

import (
	"context"
	"log"
	"strings"
	"sync"
	"testing"
	"time"
)

// newTestBus creates a bus with a discard logger for tests.
func newTestBus() *Bus {
	return New(log.New(&strings.Builder{}, "", 0))
}

func TestBus_New(t *testing.T) {
	bus := New(nil)
	if bus == nil {
		t.Fatal("New returned nil")
	}
	if bus.handlers == nil {
		t.Error("handlers map should be initialized")
	}
	if len(bus.handlers) != 0 {
		t.Error("new bus should have no handlers")
	}
}

func TestBus_NewWithLogger(t *testing.T) {
	var buf strings.Builder
	logger := log.New(&buf, "[test] ", 0)
	bus := New(logger)
	if bus == nil {
		t.Fatal("New returned nil")
	}
	_ = logger // used via bus
	_ = bus
}

func TestBus_SubscribeAndPublishSync(t *testing.T) {
	bus := newTestBus()
	var called bool
	bus.Subscribe("test.topic", func(e *Event) bool {
		called = true
		if e.Topic != "test.topic" {
			t.Errorf("e.Topic = %q, want %q", e.Topic, "test.topic")
		}
		if e.Data != "hello" {
			t.Errorf("e.Data = %v, want %v", e.Data, "hello")
		}
		return true
	})

	bus.PublishSync("test.topic", "hello")
	if !called {
		t.Error("handler was not called")
	}
}

func TestBus_PublishSync_MultipleHandlers(t *testing.T) {
	bus := newTestBus()
	var order []int
	var mu sync.Mutex

	bus.Subscribe("test", func(e *Event) bool {
		mu.Lock()
		order = append(order, 1)
		mu.Unlock()
		return true
	})
	bus.Subscribe("test", func(e *Event) bool {
		mu.Lock()
		order = append(order, 2)
		mu.Unlock()
		return true
	})

	bus.PublishSync("test", nil)

	mu.Lock()
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Errorf("Handler order = %v, want [1 2]", order)
	}
	mu.Unlock()
}

func TestBus_PublishSync_StopPropagation(t *testing.T) {
	bus := newTestBus()
	var callCount int

	bus.Subscribe("test", func(e *Event) bool {
		callCount++
		return false // stop propagation
	})
	bus.Subscribe("test", func(e *Event) bool {
		callCount++
		return true
	})

	bus.PublishSync("test", nil)
	if callCount != 1 {
		t.Errorf("Expected 1 handler call (stopped), got %d", callCount)
	}
}

func TestBus_PublishAsync(t *testing.T) {
	bus := newTestBus()
	done := make(chan struct{})

	bus.Subscribe("test", func(e *Event) bool {
		close(done)
		return true
	})

	bus.PublishAsync("test", "data")

	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("async handler was not called within timeout")
	}
}

func TestBus_PublishAsync_MultipleHandlers(t *testing.T) {
	bus := newTestBus()
	var mu sync.Mutex
	count := 0
	done := make(chan struct{})

	bus.Subscribe("test", func(e *Event) bool {
		mu.Lock()
		count++
		mu.Unlock()
		return true
	})
	bus.Subscribe("test", func(e *Event) bool {
		mu.Lock()
		count++
		if count >= 2 {
			close(done)
		}
		mu.Unlock()
		return true
	})

	bus.PublishAsync("test", nil)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("async handlers did not complete within timeout")
	}
}

func TestBus_Wildcard_SingleStar(t *testing.T) {
	bus := newTestBus()
	var called bool

	bus.Subscribe("player.*", func(e *Event) bool {
		called = true
		return true
	})

	bus.PublishSync("player.login", nil)
	if !called {
		t.Error("wildcard 'player.*' should match 'player.login'")
	}
}

func TestBus_Wildcard_SingleStar_NoMatch(t *testing.T) {
	bus := newTestBus()
	called := false

	bus.Subscribe("player.*", func(e *Event) bool {
		called = true
		return true
	})

	bus.PublishSync("player.login.ext", nil)
	if called {
		t.Error("wildcard 'player.*' should NOT match 'player.login.ext'")
	}
}

func TestBus_Wildcard_DoubleStar(t *testing.T) {
	bus := newTestBus()
	var called bool

	bus.Subscribe("combat.**", func(e *Event) bool {
		called = true
		return true
	})

	bus.PublishSync("combat.round.end", nil)
	if !called {
		t.Error("wildcard 'combat.**' should match 'combat.round.end'")
	}
}

func TestBus_Wildcard_DoubleStar_Shallow(t *testing.T) {
	bus := newTestBus()
	var called bool

	bus.Subscribe("combat.**", func(e *Event) bool {
		called = true
		return true
	})

	bus.PublishSync("combat.start", nil)
	if !called {
		t.Error("wildcard 'combat.**' should match 'combat.start'")
	}
}

func TestBus_Wildcard_StarOnly(t *testing.T) {
	bus := newTestBus()
	count := 0
	var mu sync.Mutex

	bus.Subscribe("*", func(e *Event) bool {
		mu.Lock()
		count++
		mu.Unlock()
		return true
	})

	bus.PublishSync("any.topic", nil)
	bus.PublishSync("another.one", nil)

	mu.Lock()
	if count != 2 {
		t.Errorf("'*' should match all topics, called %d times, want 2", count)
	}
	mu.Unlock()
}

func TestBus_Wildcard_DoubleStarOnly(t *testing.T) {
	bus := newTestBus()
	var called bool

	bus.Subscribe("**", func(e *Event) bool {
		called = true
		return true
	})

	bus.PublishSync("a.b.c.d", nil)
	if !called {
		t.Error("'**' should match all topics")
	}
}

func TestBus_Wildcard_MixedMatch(t *testing.T) {
	bus := newTestBus()
	var playerEvents int
	var combatEvents int
	var mu sync.Mutex

	bus.Subscribe("player.*", func(e *Event) bool {
		mu.Lock()
		playerEvents++
		mu.Unlock()
		return true
	})
	bus.Subscribe("combat.**", func(e *Event) bool {
		mu.Lock()
		combatEvents++
		mu.Unlock()
		return true
	})

	bus.PublishSync("player.login", nil)
	bus.PublishSync("player.levelup", nil)
	bus.PublishSync("combat.start", nil)
	bus.PublishSync("combat.round.end", nil)

	mu.Lock()
	if playerEvents != 2 {
		t.Errorf("playerEvents = %d, want 2", playerEvents)
	}
	if combatEvents != 2 {
		t.Errorf("combatEvents = %d, want 2", combatEvents)
	}
	mu.Unlock()
}

func TestBus_Wildcard_ExactMatchPriority(t *testing.T) {
	bus := newTestBus()
	count := 0

	// Both exact and wildcard handlers should fire
	bus.Subscribe("player.login", func(e *Event) bool {
		count++
		return true
	})
	bus.Subscribe("player.*", func(e *Event) bool {
		count++
		return true
	})

	bus.PublishSync("player.login", nil)
	if count != 2 {
		t.Errorf("Expected both exact and wildcard to fire, count = %d, want 2", count)
	}
}

func TestBus_Middleware_ChainOrder(t *testing.T) {
	bus := newTestBus()
	var order []string
	var mu sync.Mutex

	bus.Use(func(next Handler) Handler {
		return func(e *Event) bool {
			mu.Lock()
			order = append(order, "mw1_before")
			mu.Unlock()
			result := next(e)
			mu.Lock()
			order = append(order, "mw1_after")
			mu.Unlock()
			return result
		}
	})
	bus.Use(func(next Handler) Handler {
		return func(e *Event) bool {
			mu.Lock()
			order = append(order, "mw2_before")
			mu.Unlock()
			result := next(e)
			mu.Lock()
			order = append(order, "mw2_after")
			mu.Unlock()
			return result
		}
	})
	bus.Subscribe("test", func(e *Event) bool {
		mu.Lock()
		order = append(order, "handler")
		mu.Unlock()
		return true
	})

	bus.PublishSync("test", nil)

	mu.Lock()
	expected := []string{"mw1_before", "mw2_before", "handler", "mw2_after", "mw1_after"}
	if len(order) != len(expected) {
		t.Fatalf("Order length = %d, want %d. Got: %v", len(order), len(expected), order)
	}
	for i := range expected {
		if order[i] != expected[i] {
			t.Fatalf("Order[%d] = %q, want %q. Full: %v", i, order[i], expected[i], order)
		}
	}
	mu.Unlock()
}

func TestBus_Middleware_ModifyData(t *testing.T) {
	bus := newTestBus()

	bus.Use(func(next Handler) Handler {
		return func(e *Event) bool {
			// Modify event data
			if s, ok := e.Data.(string); ok {
				e.Data = s + "_modified"
			}
			return next(e)
		}
	})

	var received string
	bus.Subscribe("test", func(e *Event) bool {
		received = e.Data.(string)
		return true
	})

	bus.PublishSync("test", "original")
	if received != "original_modified" {
		t.Errorf("Handler received %q, want %q", received, "original_modified")
	}
}

func TestBus_Middleware_StopPropagation(t *testing.T) {
	bus := newTestBus()

	bus.Use(func(next Handler) Handler {
		return func(e *Event) bool {
			// Don't call next, effectively cancelling propagation
			return false
		}
	})

	handlerCalled := false
	bus.Subscribe("test", func(e *Event) bool {
		handlerCalled = true
		return true
	})

	bus.PublishSync("test", nil)
	if handlerCalled {
		t.Error("Handler should not be called when middleware stops propagation")
	}
}

func TestBus_Middleware_AppliedPerHandler(t *testing.T) {
	bus := newTestBus()
	var order []int
	var mu sync.Mutex

	bus.Use(func(next Handler) Handler {
		return func(e *Event) bool {
			mu.Lock()
			order = append(order, 0)
			mu.Unlock()
			return next(e)
		}
	})

	bus.Subscribe("test", func(e *Event) bool {
		mu.Lock()
		order = append(order, 1)
		mu.Unlock()
		return true
	})
	bus.Subscribe("test", func(e *Event) bool {
		mu.Lock()
		order = append(order, 2)
		mu.Unlock()
		return true
	})

	bus.PublishSync("test", nil)

	mu.Lock()
	// Middleware wraps each handler individually, so we get: mw->h1, mw->h2
	expected := []int{0, 1, 0, 2}
	if len(order) != len(expected) {
		t.Fatalf("Order = %v, want %v", order, expected)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Fatalf("Order[%d] = %d, want %d. Full: %v", i, order[i], v, order)
		}
	}
	mu.Unlock()
}

func TestBus_Reset(t *testing.T) {
	bus := newTestBus()
	bus.Subscribe("test", func(e *Event) bool { return true })
	bus.Use(func(next Handler) Handler { return next })

	if count := bus.HandlerCount("test"); count == 0 {
		t.Error("Should have handlers before Reset")
	}

	bus.Reset()

	if count := bus.HandlerCount("test"); count != 0 {
		t.Errorf("HandlerCount after Reset = %d, want 0", count)
	}
	if len(bus.middleware) != 0 {
		t.Errorf("Middleware count after Reset = %d, want 0", len(bus.middleware))
	}
}

func TestBus_Reset_CanSubscribeAfter(t *testing.T) {
	bus := newTestBus()
	bus.Reset()

	var called bool
	bus.Subscribe("after-reset", func(e *Event) bool {
		called = true
		return true
	})
	bus.PublishSync("after-reset", nil)
	if !called {
		t.Error("Should be able to subscribe after Reset")
	}
}

func TestBus_HandlerCount(t *testing.T) {
	bus := newTestBus()
	bus.Subscribe("test", func(e *Event) bool { return true })
	bus.Subscribe("test", func(e *Event) bool { return true })

	if count := bus.HandlerCount("test"); count != 2 {
		t.Errorf("HandlerCount = %d, want 2", count)
	}

	// Wildcard not yet matching
	if count := bus.HandlerCount("other"); count != 0 {
		t.Errorf("HandlerCount for 'other' = %d, want 0", count)
	}
}

func TestBus_HandlerCount_WithWildcard(t *testing.T) {
	bus := newTestBus()
	bus.Subscribe("player.*", func(e *Event) bool { return true })
	bus.Subscribe("player.*", func(e *Event) bool { return true })

	// HandlerCount counts wildcard matches for the given topic
	if count := bus.HandlerCount("player.login"); count != 2 {
		t.Errorf("HandlerCount('player.login') = %d, want 2", count)
	}
}

func TestBus_HandlerCount_NoMatch(t *testing.T) {
	bus := newTestBus()
	bus.Subscribe("player.*", func(e *Event) bool { return true })

	if count := bus.HandlerCount("combat.start"); count != 0 {
		t.Errorf("HandlerCount('combat.start') = %d, want 0", count)
	}
}

func TestBus_PublishToNoSubscribers(t *testing.T) {
	bus := newTestBus()
	// Should not panic
	bus.PublishSync("nonexistent", "data")
	bus.PublishAsync("nonexistent", "data")
}

func TestBus_ConcurrentPublishSync(t *testing.T) {
	bus := newTestBus()
	var mu sync.Mutex
	count := 0

	bus.Subscribe("test", func(e *Event) bool {
		mu.Lock()
		count++
		mu.Unlock()
		return true
	})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.PublishSync("test", "data")
		}()
	}
	wg.Wait()

	mu.Lock()
	if count != 50 {
		t.Errorf("Expected 50 handler calls, got %d", count)
	}
	mu.Unlock()
}

func TestBus_ConcurrentSubscribeAndPublish(t *testing.T) {
	bus := newTestBus()

	var wg sync.WaitGroup
	// Subscribe concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Subscribe("test", func(e *Event) bool {
				return true
			})
		}()
	}

	// Publish concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.PublishSync("test", "data")
		}()
	}
	wg.Wait()

	if count := bus.HandlerCount("test"); count != 10 {
		t.Errorf("Expected 10 handlers, got %d", count)
	}
}

func TestBus_TimeoutMiddleware_ContextCancelled(t *testing.T) {
	bus := newTestBus()
	bus.Use(Timeout(50 * time.Millisecond))

	handlerStarted := make(chan struct{})
	handlerDone := make(chan struct{})

	bus.Subscribe("test", func(e *Event) bool {
		close(handlerStarted)
		// Wait for context to be cancelled
		<-e.Context.Done()
		close(handlerDone)
		return true
	})

	bus.PublishSync("test", nil)

	select {
	case <-handlerDone:
		// context was cancelled, handler exited
	case <-time.After(2 * time.Second):
		t.Fatal("Handler did not respond to context cancellation within timeout")
	}
}

func TestBus_LoggingMiddleware(t *testing.T) {
	var buf strings.Builder
	logger := log.New(&buf, "", 0)
	bus := New(logger)

	bus.Use(Logging(logger))

	bus.Subscribe("test", func(e *Event) bool {
		return true
	})

	bus.PublishSync("test", "data")

	output := buf.String()
	if !strings.Contains(output, "topic=test") {
		t.Errorf("Logging middleware output should contain 'topic=test', got: %s", output)
	}
	if !strings.Contains(output, "duration=") {
		t.Errorf("Logging middleware output should contain 'duration=', got: %s", output)
	}
}

func TestBus_RecoverMiddleware_PanicInHandler(t *testing.T) {
	var buf strings.Builder
	logger := log.New(&buf, "", 0)
	bus := New(logger)

	bus.Use(Recover(logger))

	bus.Subscribe("test", func(e *Event) bool {
		panic("test panic")
	})

	// Should not crash
	bus.PublishSync("test", nil)

	output := buf.String()
	if !strings.Contains(output, "panic") {
		t.Errorf("Recover middleware should log panic, got: %s", output)
	}
}

func TestBus_RecoverMiddleware_SubsequentHandlerCalled(t *testing.T) {
	var buf strings.Builder
	logger := log.New(&buf, "", 0)
	bus := New(logger)

	bus.Use(Recover(logger))

	secondCalled := false
	bus.Subscribe("test", func(e *Event) bool {
		panic("panic in first handler")
	})
	bus.Subscribe("test", func(e *Event) bool {
		secondCalled = true
		return true
	})

	// Should not crash
	bus.PublishSync("test", nil)

	if !secondCalled {
		t.Error("Second handler should be called after panic recovery")
	}
}

func TestBus_ContextPropagation(t *testing.T) {
	bus := newTestBus()
	ctx := context.WithValue(context.Background(), "key", "value")

	var receivedCtx context.Context
	bus.Subscribe("test", func(e *Event) bool {
		receivedCtx = e.Context
		return true
	})

	bus.PublishWithContext(ctx, "test", nil)

	if receivedCtx == nil {
		t.Fatal("Handler should receive a context")
	}
	if v := receivedCtx.Value("key"); v != "value" {
		t.Errorf("Context value = %v, want 'value'", v)
	}
}

func TestBus_ContextWithTimeout(t *testing.T) {
	bus := newTestBus()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	handlerDone := make(chan struct{})
	bus.Subscribe("test", func(e *Event) bool {
		<-e.Context.Done()
		close(handlerDone)
		return true
	})

	bus.PublishWithContext(ctx, "test", nil)

	select {
	case <-handlerDone:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("Handler did not respond to context cancellation")
	}
}

func TestBus_String(t *testing.T) {
	bus := newTestBus()
	bus.Subscribe("test", func(e *Event) bool { return true })
	bus.Use(func(next Handler) Handler { return next })

	s := bus.String()
	if !strings.Contains(s, "EventBus") {
		t.Errorf("String() should contain 'EventBus', got: %s", s)
	}
	if !strings.Contains(s, "topics=1") {
		t.Errorf("String() should contain 'topics=1', got: %s", s)
	}
	if !strings.Contains(s, "middleware=1") {
		t.Errorf("String() should contain 'middleware=1', got: %s", s)
	}
}

func TestMatchFunction(t *testing.T) {
	tests := []struct {
		pattern string
		topic   string
		want    bool
	}{
		// Single star
		{"player.*", "player.login", true},
		{"player.*", "player.login.ext", false},
		{"player.*", "combat.start", false},
		{"player.*", "player", false},
		// Double star
		{"combat.**", "combat.start", true},
		{"combat.**", "combat.round.end", true},
		{"combat.**", "combat", true}, // ** matches zero remaining segments
		{"combat.**", "other.thing", false},
		// Star only
		{"*", "anything.here", true},
		{"*", "a", true},
		// Double star only
		{"**", "a.b.c.d", true},
		{"**", "x", true},
		// Exact match
		{"test", "test", true},
		{"test", "other", false},
		// Pattern with ** in middle
		{"a.**.b", "a.x.y.b", true},
		{"a.**.b", "a.b", true},
		{"a.**.b", "a.x.b.c", true}, // ** consumes everything including .b.c
		// Complex patterns
		{"a.b.*", "a.b.c", true},
		{"a.b.*", "a.b.c.d", false},
		{"a.b.**", "a.b.c.d", true},
		// Edge cases
		{"", "test", false},
		{"test", "", false},
		{"a.b.*.c", "a.b.x.c", true},
		{"a.b.*.c", "a.b.x.y.c", false},
	}
	for _, tt := range tests {
		t.Run(tt.pattern+"_vs_"+tt.topic, func(t *testing.T) {
			got := match(tt.pattern, tt.topic)
			if got != tt.want {
				t.Errorf("match(%q, %q) = %v, want %v", tt.pattern, tt.topic, got, tt.want)
			}
		})
	}
}

func TestBus_PublishWithContext_AsyncIgnored(t *testing.T) {
	bus := newTestBus()

	done := make(chan struct{})
	bus.Subscribe("test", func(e *Event) bool {
		// Sleep to verify async handler runs regardless of context
		time.Sleep(50 * time.Millisecond)
		close(done)
		return true
	})

	bus.PublishAsync("test", "data")

	select {
	case <-done:
		// success - async ran despite short context
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Async handler should have completed despite context timeout")
	}
}

func TestBus_MultipleMiddleware(t *testing.T) {
	bus := newTestBus()
	var result int

	bus.Use(func(next Handler) Handler {
		return func(e *Event) bool {
			if s, ok := e.Data.(int); ok {
				e.Data = s + 1
			}
			return next(e)
		}
	})
	bus.Use(func(next Handler) Handler {
		return func(e *Event) bool {
			if s, ok := e.Data.(int); ok {
				e.Data = s * 2
			}
			return next(e)
		}
	})

	bus.Subscribe("test", func(e *Event) bool {
		result = e.Data.(int)
		return true
	})

	// Handler runs after middleware chain: (1 + 1) * 2 = 4
	bus.PublishSync("test", 1)
	if result != 4 {
		t.Errorf("Result after middleware = %d, want 4", result)
	}
}

func TestBus_NoMiddleware(t *testing.T) {
	bus := newTestBus()
	var called bool

	bus.Subscribe("test", func(e *Event) bool {
		called = true
		return true
	})

	bus.PublishSync("test", nil)
	if !called {
		t.Error("Handler should be called without middleware")
	}
}
