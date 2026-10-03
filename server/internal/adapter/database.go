package adapter

import (
	"fmt"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// dbPath is the default path to the SQLite database file.
const dbPath = "database.db"

// NewDB initializes and returns a new GORM database connection using SQLite.
func NewDB() (*gorm.DB, error) {
	if err := ensureFile(); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// Migrate the schema
	if err := db.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

// ensureFile ensures the database file exists with restricted permissions (0600).
func ensureFile() error {
	f, err := os.OpenFile(dbPath, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("create database file: %w", err)
	}
	return f.Close()
}
