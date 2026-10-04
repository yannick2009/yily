package adapter

import (
	"os"

	"github.com/yannick2009/yily/internal/domain/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	DatabaseFile = "database.db" // DatabaseFile is the name of the SQLite database file.
)

// NewDB initializes and returns a new GORM database connection using SQLite.
// It also performs automatic schema migration for the defined models.
// The database connection is configured to enable foreign key support, set a busy timeout, and use WAL mode for better concurrency.
func NewDB() *gorm.DB {
	if err := ensureDBFile(); err != nil {
		panic("failed to create database file")
	}

	db, err := gorm.Open(sqlite.Open(DatabaseFile+"?_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	err = db.AutoMigrate(
		&model.Project{},         // Migrate the Project model
		&model.Environment{},     // Migrate the Environment model
		&model.Variable{},        // Migrate the Variable model
		&model.VariableVersion{}, // Migrate the VariableVersion model
		&model.Tag{},             // Migrate the Tag model
	)
	if err != nil {
		panic("failed to migrate database schema")
	}

	return db
}

// ensureDBFile ensures the database file exists with 0600 permissions.
func ensureDBFile() error {
	f, err := os.OpenFile(DatabaseFile, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	return f.Close()
}
