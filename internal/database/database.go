package database

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	config "github.com/monorkin/gnome-desktop-air-monitor/internal/config"
)

var (
	DB      *gorm.DB
	once    sync.Once
	initErr error
)

func Init() error {
	once.Do(func() {
		DB, initErr = SetupDatabase()
	})
	return initErr
}

func SetupDatabase() (*gorm.DB, error) {
	dbPath := config.DBPath()

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db.Exec("PRAGMA foreign_keys = ON")

	err = Migrate(db)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}
