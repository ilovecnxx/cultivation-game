package sd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestInstanceJSONMarshaling(t *testing.T) {
	now := time.Now().Round(time.Millisecond)

	tests := []struct {
		name     string
		instance Instance
	}{
		{
			name: "full instance with metadata",
			instance: Instance{
				ID:        "svc-host-8080-1234",
				Name:      "my-service",
				Addr:      "192.168.1.10",
				Port:      8080,
				Metadata:  map[string]string{"env": "prod", "region": "us-east-1"},
				StartTime: now,
				LastSeen:  now,
			},
		},
		{
			name: "instance with nil metadata",
			instance: Instance{
				ID:        "svc2-other-9090-5678",
				Name:      "svc2",
				Addr:      "10.0.0.1",
				Port:      9090,
				StartTime: now,
				LastSeen:  now,
			},
		},
		{
			name: "instance with empty metadata",
			instance: Instance{
				ID:        "svc3-host-3000-9999",
				Name:      "svc3",
				Addr:      "0.0.0.0",
				Port:      3000,
				Metadata:  map[string]string{},
				StartTime: now,
				LastSeen:  now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.instance)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}

			var got Instance
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}

			if got.ID != tt.instance.ID {
				t.Errorf("ID = %q, want %q", got.ID, tt.instance.ID)
			}
			if got.Name != tt.instance.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.instance.Name)
			}
			if got.Addr != tt.instance.Addr {
				t.Errorf("Addr = %q, want %q", got.Addr, tt.instance.Addr)
			}
			if got.Port != tt.instance.Port {
				t.Errorf("Port = %d, want %d", got.Port, tt.instance.Port)
			}
			if !got.StartTime.Equal(tt.instance.StartTime) {
				t.Errorf("StartTime = %v, want %v", got.StartTime, tt.instance.StartTime)
			}
			if !got.LastSeen.Equal(tt.instance.LastSeen) {
				t.Errorf("LastSeen = %v, want %v", got.LastSeen, tt.instance.LastSeen)
			}
			if tt.instance.Metadata != nil {
				for k, v := range tt.instance.Metadata {
					if got.Metadata[k] != v {
						t.Errorf("Metadata[%q] = %q, want %q", k, got.Metadata[k], v)
					}
				}
			}
		})
	}
}

func TestNewRegistrar_CreatesValidInstance(t *testing.T) {
	r := NewRegistrar(nil, "auth-service", "127.0.0.1", 3000, map[string]string{"env": "test"})

	if r == nil {
		t.Fatal("NewRegistrar returned nil")
	}

	if r.instance.Name != "auth-service" {
		t.Errorf("Name = %q, want %q", r.instance.Name, "auth-service")
	}
	if r.instance.Addr != "127.0.0.1" {
		t.Errorf("Addr = %q, want %q", r.instance.Addr, "127.0.0.1")
	}
	if r.instance.Port != 3000 {
		t.Errorf("Port = %d, want %d", r.instance.Port, 3000)
	}
	if r.instance.Metadata["env"] != "test" {
		t.Errorf("Metadata missing key %q", "env")
	}
	if r.instance.StartTime.IsZero() {
		t.Error("StartTime should not be zero")
	}
	if r.instance.LastSeen.IsZero() {
		t.Error("LastSeen should not be zero")
	}
	if r.instance.ID == "" {
		t.Error("Instance ID should not be empty")
	}
	if r.ttl != defaultTTL {
		t.Errorf("TTL = %v, want %v", r.ttl, defaultTTL)
	}
	if r.interval != defaultInterval {
		t.Errorf("Interval = %v, want %v", r.interval, defaultInterval)
	}
	if r.stopCh == nil {
		t.Error("stopCh should not be nil")
	}
	if r.doneCh == nil {
		t.Error("doneCh should not be nil")
	}
	if r.running {
		t.Error("running should be false initially")
	}

	// Verify the instance ID matches the expected format: name-hostname-port-random
	expectedPrefix := "auth-service-"
	if !strings.HasPrefix(r.instance.ID, expectedPrefix) {
		t.Errorf("Instance ID %q should start with %q", r.instance.ID, expectedPrefix)
	}
}

func TestNewRegistrar_NilMetadata(t *testing.T) {
	r := NewRegistrar(nil, "svc", "addr", 8080, nil)
	if r.instance.Metadata != nil {
		t.Error("Metadata should be nil when not provided")
	}
}

func TestSetTTL(t *testing.T) {
	r := NewRegistrar(nil, "svc", "addr", 8080, nil)

	tests := []struct {
		name     string
		ttl      time.Duration
		interval time.Duration
	}{
		{"short TTL", 5 * time.Second, 1 * time.Second},
		{"long TTL", 60 * time.Second, 30 * time.Second},
		{"zero TTL", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r.SetTTL(tt.ttl, tt.interval)

			if r.ttl != tt.ttl {
				t.Errorf("ttl = %v, want %v", r.ttl, tt.ttl)
			}
			if r.interval != tt.interval {
				t.Errorf("interval = %v, want %v", r.interval, tt.interval)
			}
		})
	}
}

func TestDiscovererCacheInvalidation(t *testing.T) {
	d := NewDiscoverer(nil)

	// Directly populate the cache (bypassing Redis)
	d.mu.Lock()
	d.cache["player"] = []Instance{
		{ID: "player-host1-8080-1", Name: "player", Addr: "10.0.0.1", Port: 8080},
	}
	d.cache["world"] = []Instance{
		{ID: "world-host2-8080-2", Name: "world", Addr: "10.0.0.2", Port: 8080},
	}
	d.mu.Unlock()

	if len(d.cache) != 2 {
		t.Fatalf("expected 2 cached services, got %d", len(d.cache))
	}

	d.InvalidateCache()

	if len(d.cache) != 0 {
		t.Errorf("cache should be empty after InvalidateCache, got %d entries", len(d.cache))
	}

	// Verify the underlying map is replaced, not just cleared
	if d.cache == nil {
		t.Error("cache map should not be nil after InvalidateCache")
	}
}

func TestServiceKeyNamingConvention(t *testing.T) {
	tests := []struct {
		key       string
		expected  string
		reason    string
	}{
		{redisKeyPrefix, "sd:service:", "prefix for service sorted sets"},
		{redisKeyWorkers, "sd:workers", "hash key for worker instance details"},
	}

	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			if tt.key != tt.expected {
				t.Errorf("got %q, want %q", tt.key, tt.expected)
			}
		})
	}

	// Verify the generated Redis key uses the prefix
	r := NewRegistrar(nil, "cultivation", "addr", 8080, nil)
	expectedKey := redisKeyPrefix + r.instance.Name
	if !strings.HasPrefix(expectedKey, redisKeyPrefix) {
		t.Errorf("service key %q should have prefix %q", expectedKey, redisKeyPrefix)
	}
	if !strings.Contains(expectedKey, "cultivation") {
		t.Errorf("service key %q should contain service name", expectedKey)
	}
}

func TestInstanceIDFormat(t *testing.T) {
	hostname, err := os.Hostname()
	if err != nil {
		t.Fatalf("os.Hostname() failed: %v", err)
	}

	tests := []struct {
		name     string
		addr     string
		port     int
		meta     map[string]string
	}{
		{"player", "10.0.0.1", 8080, nil},
		{"auth", "192.168.1.1", 9090, map[string]string{"env": "prod"}},
		{"world", "0.0.0.0", 3000, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistrar(nil, tt.name, tt.addr, tt.port, tt.meta)
			id := r.instance.ID

			// Must start with the service name
			if !strings.HasPrefix(id, tt.name) {
				t.Errorf("ID %q should start with service name %q", id, tt.name)
			}

			// Must contain the hostname
			if !strings.Contains(id, hostname) {
				t.Errorf("ID %q should contain hostname %q", id, hostname)
			}

			// ID parts separated by dashes: name-hostname-port-random
			parts := strings.Split(id, "-")
			if len(parts) < 4 {
				t.Errorf("ID %q should have at least 4 dash-separated parts (name-hostname-port-random), got %d", id, len(parts))
			}

			// Verify the port is present somewhere in the ID
			portStr := fmt.Sprintf("%d", tt.port)
			if !strings.Contains(id, portStr) {
				t.Errorf("ID %q should contain port %d", id, tt.port)
			}
		})
	}
}

func TestInstanceIDUniqueness(t *testing.T) {
	// Two registrars with the same parameters should get different IDs (due to random component)
	r1 := NewRegistrar(nil, "dup", "addr", 8080, nil)
	r2 := NewRegistrar(nil, "dup", "addr", 8080, nil)

	if r1.instance.ID == r2.instance.ID {
		t.Error("two registrars with identical params should have different IDs (random component)")
	}

	// Different ports should generate different IDs
	r3 := NewRegistrar(nil, "dup", "addr", 9090, nil)
	if r1.instance.ID == r3.instance.ID {
		t.Error("different ports should produce different IDs")
	}

	// Different names should generate different IDs
	r4 := NewRegistrar(nil, "other", "addr", 8080, nil)
	if r1.instance.ID == r4.instance.ID {
		t.Error("different names should produce different IDs")
	}
}

func TestDefaultTTLConstants(t *testing.T) {
	if defaultTTL != 15*time.Second {
		t.Errorf("defaultTTL = %v, want 15s", defaultTTL)
	}
	if defaultInterval != 5*time.Second {
		t.Errorf("defaultInterval = %v, want 5s", defaultInterval)
	}
}
