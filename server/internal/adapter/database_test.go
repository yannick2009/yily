package adapter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	db, err := NewDB(path)
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })

	// The database file must be readable by its owner only.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file permissions = %o, want 600", perm)
	}

	// Connection parameters must be applied.
	var foreignKeys int
	if err := db.Raw("PRAGMA foreign_keys").Scan(&foreignKeys).Error; err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Error("foreign keys are not enabled")
	}

	var journalMode string
	if err := db.Raw("PRAGMA journal_mode").Scan(&journalMode).Error; err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("journal_mode = %q, want wal", journalMode)
	}
}

func TestNewDBInvalidPath(t *testing.T) {
	if _, err := NewDB(filepath.Join(t.TempDir(), "missing-dir", "test.db")); err == nil {
		t.Fatal("expected an error for a path in a non-existent directory")
	}
}
