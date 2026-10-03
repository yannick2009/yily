package adapter

import (
	"fmt"
	"net/url"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NewDB opens (and creates if needed) the SQLite database at path, applies
// the connection settings and checks that the database is reachable.
func NewDB(path string) (*gorm.DB, error) {

	if err := ensureFile(path); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(dsn(path)), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database %q: %w", path, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get database handle: %w", err)
	}

	// Migrate the schema
	// db.AutoMigrate()

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping database %q: %w", path, err)
	}
	return db, nil
}

// ensureFile creates the database file with owner-only permissions (0600)
// if it does not exist yet.
//
// SQLite would otherwise create it with the default umask (usually 0644),
// making it readable by every user on the machine. The -wal and -shm files
// created later by SQLite inherit the permissions of the main file.
// An existing file keeps its current permissions.
func ensureFile(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("create database file %q: %w", path, err)
	}
	return f.Close()
}

// dsn builds the go-sqlite3 connection string. Parameters are applied to
// every connection of the pool:
//   - _foreign_keys=on: SQLite does NOT enforce foreign keys by default.
//   - _journal_mode=WAL: readers do not block the writer, and vice versa.
//   - _busy_timeout=5000: wait up to 5s for a lock instead of failing at once
//     with "database is locked" under concurrent requests.
func dsn(path string) string {
	params := url.Values{}
	params.Set("_foreign_keys", "on")
	params.Set("_journal_mode", "WAL")
	params.Set("_busy_timeout", "5000")
	return path + "?" + params.Encode()
}
