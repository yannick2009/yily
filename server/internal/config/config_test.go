package config

import (
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("YILY_MASTER_KEY", "some-key")
	t.Setenv("YILY_ADDR", "")
	t.Setenv("YILY_DB_PATH", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Addr != defaultAddr || cfg.DBPath != defaultDBPath {
		t.Fatalf("defaults not applied: %+v", cfg)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("YILY_MASTER_KEY", "  some-key\n") // trailing newline from .env files
	t.Setenv("YILY_ADDR", "127.0.0.1:9000")
	t.Setenv("YILY_DB_PATH", "/tmp/test.db")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MasterKey != "some-key" || cfg.Addr != "127.0.0.1:9000" || cfg.DBPath != "/tmp/test.db" {
		t.Fatalf("env not applied: %+v", cfg)
	}
}

func TestLoadRequiresMasterKey(t *testing.T) {
	for _, v := range []string{"", "   "} {
		t.Setenv("YILY_MASTER_KEY", v)
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "YILY_MASTER_KEY") {
			t.Fatalf("value %q: want master key error, got %v", v, err)
		}
	}
}

func TestStringRedactsMasterKey(t *testing.T) {
	cfg := Config{MasterKey: "super-secret"}
	if strings.Contains(cfg.String(), "super-secret") {
		t.Fatal("master key leaked by String()")
	}
}
