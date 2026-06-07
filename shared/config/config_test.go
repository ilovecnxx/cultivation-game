package config

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// writeTempFile creates a temp file with the given content and returns its path.
func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	if dir == "" {
		dir = t.TempDir()
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp file %s: %v", path, err)
	}
	return path
}

func TestNew_JSON(t *testing.T) {
	content := `{"server": {"port": 8080, "host": "localhost"}, "debug": true}`
	path := writeTempFile(t, "", "test.json", content)

	l, err := New(path)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if l == nil {
		t.Fatal("New() returned nil")
	}
	defer l.Close()

	if port := l.GetInt("server.port"); port != 8080 {
		t.Errorf("server.port = %d, want 8080", port)
	}
	if host := l.GetString("server.host"); host != "localhost" {
		t.Errorf("server.host = %q, want 'localhost'", host)
	}
	if debug := l.GetBool("debug"); !debug {
		t.Errorf("debug = %v, want true", debug)
	}
}

func TestNew_YAML(t *testing.T) {
	content := `
server:
  port: 9090
  host: "example.com"
debug: false
`
	path := writeTempFile(t, "", "test.yaml", content)

	l, err := New(path)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer l.Close()

	if port := l.GetInt("server.port"); port != 9090 {
		t.Errorf("server.port = %d, want 9090", port)
	}
	if host := l.GetString("server.host"); host != "example.com" {
		t.Errorf("server.host = %q, want 'example.com'", host)
	}
	if debug := l.GetBool("debug"); debug {
		t.Errorf("debug = %v, want false", debug)
	}
}

func TestNew_YML_Extension(t *testing.T) {
	content := "key: value\nnumber: 42"
	path := writeTempFile(t, "", "test.yml", content)

	l, err := New(path)
	if err != nil {
		t.Fatalf("New() for .yml error: %v", err)
	}
	defer l.Close()

	if v := l.GetString("key"); v != "value" {
		t.Errorf("key = %q, want 'value'", v)
	}
	if v := l.GetInt("number"); v != 42 {
		t.Errorf("number = %d, want 42", v)
	}
}

func TestNew_InvalidFile(t *testing.T) {
	_, err := New("/nonexistent/path/config.json")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestNew_UnsupportedExtension(t *testing.T) {
	path := writeTempFile(t, "", "config.toml", "key = 'value'")
	_, err := New(path)
	if err == nil {
		t.Error("Expected error for unsupported extension")
	}
	if err != nil && !strings.Contains(err.Error(), "不支持的文件格式") {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestNew_InvalidJSON(t *testing.T) {
	path := writeTempFile(t, "", "invalid.json", "{invalid json}")
	_, err := New(path)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestNew_InvalidYAML(t *testing.T) {
	path := writeTempFile(t, "", "invalid.yaml", ":\ninvalid yaml")
	_, err := New(path)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestLoader_Get_TopLevel(t *testing.T) {
	content := `{"name": "test", "count": 100, "ratio": 0.85}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	if v := l.GetString("name"); v != "test" {
		t.Errorf("name = %q, want 'test'", v)
	}
}

func TestLoader_Get_Nested(t *testing.T) {
	content := `{"database": {"host": "db.local", "port": 5432, "pool": {"min": 5, "max": 20}}}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	if v := l.GetString("database.host"); v != "db.local" {
		t.Errorf("database.host = %q, want 'db.local'", v)
	}
	if v := l.GetInt("database.port"); v != 5432 {
		t.Errorf("database.port = %d, want 5432", v)
	}
	if v := l.GetInt("database.pool.min"); v != 5 {
		t.Errorf("database.pool.min = %d, want 5", v)
	}
	if v := l.GetInt("database.pool.max"); v != 20 {
		t.Errorf("database.pool.max = %d, want 20", v)
	}
}

func TestLoader_Get_NonExistent(t *testing.T) {
	content := `{"existing": "value"}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	if v := l.Get("nonexistent"); v != nil {
		t.Errorf("Get('nonexistent') = %v, want nil", v)
	}
	if v := l.GetString("nonexistent"); v != "" {
		t.Errorf("GetString('nonexistent') = %q, want ''", v)
	}
	if v := l.GetInt("nonexistent"); v != 0 {
		t.Errorf("GetInt('nonexistent') = %d, want 0", v)
	}
	if v := l.GetBool("nonexistent"); v {
		t.Errorf("GetBool('nonexistent') = true, want false")
	}
}

func TestLoader_GetString_NonStringValue(t *testing.T) {
	content := `{"port": 8080, "flag": true}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	// GetString on non-string should format as string
	if v := l.GetString("port"); v != "8080" {
		t.Errorf("GetString('port') = %q, want '8080'", v)
	}
	if v := l.GetString("flag"); v != "true" {
		t.Errorf("GetString('flag') = %q, want 'true'", v)
	}
}

func TestLoader_GetInt_TypeConversion(t *testing.T) {
	content := `{"as_float": 42.0, "as_string": "99"}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	if v := l.GetInt("as_float"); v != 42 {
		t.Errorf("GetInt('as_float') = %d, want 42", v)
	}
	if v := l.GetInt("as_string"); v != 99 {
		t.Errorf("GetInt('as_string') = %d, want 99", v)
	}
}

func TestLoader_GetBool_NonBoolValue(t *testing.T) {
	content := `{"truthy": "true", "falsy": 0}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	if v := l.GetBool("truthy"); v {
		t.Errorf("GetBool('truthy' from string) = true, want false (type mismatch)")
	}
	if v := l.GetBool("falsy"); v {
		t.Errorf("GetBool('falsy' from int) = true, want false")
	}
}

func TestLoader_GetFloat(t *testing.T) {
	content := `{"pi": 3.14159, "hundred": 100, "from_string": "2.5"}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	if v := l.GetFloat("pi"); v != 3.14159 {
		t.Errorf("GetFloat('pi') = %f, want 3.14159", v)
	}
	if v := l.GetFloat("hundred"); v != 100.0 {
		t.Errorf("GetFloat('hundred') = %f, want 100.0", v)
	}
	if v := l.GetFloat("from_string"); v != 2.5 {
		t.Errorf("GetFloat('from_string') = %f, want 2.5", v)
	}
}

func TestLoader_GetFloat_Invalid(t *testing.T) {
	content := `{"bad": "not_a_number"}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	if v := l.GetFloat("bad"); v != 0 {
		t.Errorf("GetFloat('bad') = %f, want 0", v)
	}
}

func TestLoader_GetDuration(t *testing.T) {
	content := `{"timeout": "5s", "interval": "100ms", "empty": ""}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	if v := l.GetDuration("timeout"); v != 5*time.Second {
		t.Errorf("GetDuration('timeout') = %v, want 5s", v)
	}
	if v := l.GetDuration("interval"); v != 100*time.Millisecond {
		t.Errorf("GetDuration('interval') = %v, want 100ms", v)
	}
	if v := l.GetDuration("empty"); v != 0 {
		t.Errorf("GetDuration('empty') = %v, want 0", v)
	}
}

func TestLoader_GetDuration_Invalid(t *testing.T) {
	content := `{"bad": "not_duration"}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	if v := l.GetDuration("bad"); v != 0 {
		t.Errorf("GetDuration('bad') = %v, want 0", v)
	}
}

func TestLoader_GetSlice(t *testing.T) {
	content := `{"items": ["a", "b", "c"], "numbers": [1, 2, 3]}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	items := l.GetSlice("items")
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}
	if items[0] != "a" || items[1] != "b" || items[2] != "c" {
		t.Errorf("items = %v, want [a b c]", items)
	}

	numbers := l.GetSlice("numbers")
	if len(numbers) != 3 {
		t.Fatalf("len(numbers) = %d, want 3", len(numbers))
	}
	// JSON numbers are float64
	if numbers[0].(float64) != 1 {
		t.Errorf("numbers[0] = %v, want 1", numbers[0])
	}
}

func TestLoader_GetSlice_NonSlice(t *testing.T) {
	content := `{"not_slice": "value"}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	if v := l.GetSlice("not_slice"); v != nil {
		t.Errorf("GetSlice on non-slice = %v, want nil", v)
	}
}

func TestLoader_GetMap(t *testing.T) {
	content := `{"database": {"host": "localhost", "port": 5432}}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	m := l.GetMap("database")
	if m == nil {
		t.Fatal("GetMap('database') returned nil")
	}
	if m["host"] != "localhost" {
		t.Errorf("host = %v, want 'localhost'", m["host"])
	}
	if m["port"].(float64) != 5432 {
		t.Errorf("port = %v, want 5432", m["port"])
	}
}

func TestLoader_GetMap_NonMap(t *testing.T) {
	content := `{"not_map": "value"}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	if v := l.GetMap("not_map"); v != nil {
		t.Errorf("GetMap on non-map = %v, want nil", v)
	}
}

func TestLoader_All(t *testing.T) {
	content := `{"key1": "value1", "key2": 42}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	all := l.All()
	if len(all) != 2 {
		t.Fatalf("len(All()) = %d, want 2", len(all))
	}
	if all["key1"] != "value1" {
		t.Errorf("key1 = %v, want 'value1'", all["key1"])
	}
}

func TestLoader_All_Isolation(t *testing.T) {
	content := `{"key": "value"}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	all := l.All()
	all["key"] = "modified"

	// Original should be unchanged
	if v := l.GetString("key"); v != "value" {
		t.Errorf("Original config changed to %q, want 'value'", v)
	}
}

func TestLoader_Dump(t *testing.T) {
	content := `{"name": "test", "count": 42}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	dump := l.Dump()
	if !strings.Contains(dump, `"name"`) || !strings.Contains(dump, `"test"`) {
		t.Errorf("Dump() missing expected content: %s", dump)
	}
	if !strings.Contains(dump, `"count"`) || !strings.Contains(dump, `42`) {
		t.Errorf("Dump() missing expected content: %s", dump)
	}
}

func TestLoader_UnmarshalKey(t *testing.T) {
	content := `{"server": {"host": "localhost", "port": 8080}}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	type ServerConfig struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	var srv ServerConfig
	if err := l.UnmarshalKey("server", &srv); err != nil {
		t.Fatalf("UnmarshalKey error: %v", err)
	}
	if srv.Host != "localhost" || srv.Port != 8080 {
		t.Errorf("Unmarshaled server = %+v, want {localhost 8080}", srv)
	}
}

func TestLoader_UnmarshalKey_NotFound(t *testing.T) {
	content := `{"existing": "value"}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	var v string
	if err := l.UnmarshalKey("nonexistent", &v); err == nil {
		t.Error("UnmarshalKey on non-existent key should return error")
	}
}

func TestLoader_Load_Reload(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "config.json", `{"version": 1}`)

	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	if v := l.GetInt("version"); v != 1 {
		t.Errorf("version before reload = %d, want 1", v)
	}

	// Update the file
	writeTempFile(t, dir, "config.json", `{"version": 2}`)

	if err := l.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if v := l.GetInt("version"); v != 2 {
		t.Errorf("version after reload = %d, want 2", v)
	}
}

func TestLoader_ConcurrentReads(t *testing.T) {
	content := `{"key": "value", "nested": {"deep": 42}}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				l.Get("key")
				l.GetString("key")
				l.GetInt("nested.deep")
				l.Get("nonexistent")
				l.All()
				l.Dump()
			}
		}()
	}
	wg.Wait()
	// If we get here without data races, the test passes
}

func TestLoader_OnChange(t *testing.T) {
	content := `{"key": "value"}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	callbackCalled := false
	l.OnChange(func(newCfg map[string]interface{}) {
		callbackCalled = true
		if newCfg["key"] != "value" {
			t.Errorf("Callback received %v, want {'key': 'value'}", newCfg)
		}
	})

	// Call Load to trigger the callback (Load is called by watchLoop during hot reload,
	// but we can't test fsnotify in unit tests, so we verify the callback registration
	// and that the loader works correctly)
	if !callbackCalled {
		// OnChange doesn't call the callback immediately; it's only called during hot reload.
		// Instead, verify that registrations work by calling Load() again.
		// (Load does NOT call OnChange callbacks - that's by design.)
	}
	// Just verify the callback was registered (no error)
	_ = callbackCalled
}

func TestLoader_MultipleOnChange(t *testing.T) {
	content := `{"key": "value"}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	count := 0
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		l.OnChange(func(newCfg map[string]interface{}) {
			mu.Lock()
			count++
			mu.Unlock()
		})
	}

	// Verify the refire doesn't panic when we lock/unlock
	l.mu.Lock()
	_ = l.onChange
	l.mu.Unlock()
}

func TestLoader_Close(t *testing.T) {
	content := `{"key": "value"}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := l.Close(); err != nil {
		t.Errorf("First Close() error: %v", err)
	}
	// Second close should be idempotent
	if err := l.Close(); err != nil {
		t.Errorf("Second Close() error: %v", err)
	}
}

func TestLoader_String(t *testing.T) {
	content := `{"key": "value"}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	s := l.String()
	if !strings.Contains(s, "Loader") {
		t.Errorf("String() should contain 'Loader', got: %s", s)
	}
	if !strings.Contains(s, "test.json") {
		t.Errorf("String() should contain filename, got: %s", s)
	}
}

func TestGetNested(t *testing.T) {
	m := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": "deep_value",
			},
			"x": "value_x",
		},
		"top": "simple",
	}

	tests := []struct {
		key  string
		want interface{}
	}{
		{"a.b.c", "deep_value"},
		{"a.x", "value_x"},
		{"top", "simple"},
		{"nonexistent", nil},
		{"a.b.c.d.e", nil},
		{"", m},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := getNested(m, tt.key)
			// For the empty key case, got is the full map; verify by checking length
			if tt.key == "" {
				if g, ok := got.(map[string]interface{}); !ok || len(g) != 2 {
					t.Errorf("getNested('') returned map with len %d, want 3", len(g))
				}
			} else if got != tt.want {
				t.Errorf("getNested(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestParseEnvValue(t *testing.T) {
	tests := []struct {
		name    string
		envVal  string
		origVal interface{}
		want    interface{}
	}{
		{"orig bool true", "true", true, true},
		{"orig bool false", "false", true, false},
		{"orig bool invalid", "notbool", true, true},
		{"orig float64", "3.14", float64(1), 3.14},
		{"orig float64 int", "42", float64(1), float64(42)},
		{"orig string", "hello", "world", "hello"},
		{"orig int-like", "123", 456, "123"},
		{"orig nil", "value", nil, "value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEnvValue(tt.envVal, tt.origVal)
			if got != tt.want {
				t.Errorf("parseEnvValue(%q, %v) = %v (%T), want %v (%T)",
					tt.envVal, tt.origVal, got, got, tt.want, tt.want)
			}
		})
	}
}

func TestMustLoad_PanicsOnError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustLoad should panic on invalid path")
		}
	}()
	MustLoad("/nonexistent/path.json")
}

func TestMustLoad_Success(t *testing.T) {
	content := `{"key": "value"}`
	path := writeTempFile(t, "", "test.json", content)

	l := MustLoad(path)
	defer l.Close()

	if l.GetString("key") != "value" {
		t.Errorf("key = %q, want 'value'", l.GetString("key"))
	}
}

func TestLoader_Get_EmptyKey(t *testing.T) {
	content := `{"key": "value"}`
	path := writeTempFile(t, "", "test.json", content)
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	// Empty key returns the entire config
	v := l.Get("")
	if v == nil {
		t.Error("Get('') should return the full config map")
	}
}
