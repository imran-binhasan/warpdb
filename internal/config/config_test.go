package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.Port != 6379 {
		t.Fatalf("expected port 6379 got %d", cfg.Port)
	}
	if cfg.WALPath != "wal" {
		t.Fatalf("expected wal_path 'wal' got '%s'", cfg.WALPath)
	}
	if cfg.MaxClients != 10000 {
		t.Fatalf("expected maxclients 10000 got %d", cfg.MaxClients)
	}
	if cfg.WalMaxSizeMB != 64 {
		t.Fatalf("expected wal_max_size_mb 64 got %d", cfg.WalMaxSizeMB)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "warpdb.json")
	content := `{"port": 6380, "requirepass": "secret", "maxclients": 500}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 6380 {
		t.Fatalf("expected port 6380 got %d", cfg.Port)
	}
	if cfg.RequirePass != "secret" {
		t.Fatalf("expected requirepass 'secret' got '%s'", cfg.RequirePass)
	}
	if cfg.MaxClients != 500 {
		t.Fatalf("expected maxclients 500 got %d", cfg.MaxClients)
	}
	if cfg.WALPath != "wal" {
		t.Fatalf("expected default wal_path 'wal' got '%s'", cfg.WALPath)
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/config.json")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 6379 {
		t.Fatalf("expected default port 6379 got %d", cfg.Port)
	}
}

func TestLogLevelFromString(t *testing.T) {
	tests := []struct {
		input string
		level int
	}{
		{"debug", -4},
		{"info", 0},
		{"warn", 4},
		{"error", 8},
		{"something", 0},
	}
	for _, tc := range tests {
		lvl := LogLevelFromString(tc.input)
		if int(lvl) != tc.level {
			t.Fatalf("LogLevelFromString(%q) = %d, want %d", tc.input, lvl, tc.level)
		}
	}
}

func TestTimeout(t *testing.T) {
	cfg := Config{TimeoutSeconds: 60}
	if cfg.Timeout().Seconds() != 60 {
		t.Fatalf("expected 60s timeout got %v", cfg.Timeout())
	}
	cfg.TimeoutSeconds = 0
	if cfg.Timeout() != 0 {
		t.Fatal("expected zero timeout")
	}
}
