package config

import (
	"os"
	"path/filepath"
)

const (
	DB_NAME = "database.sqlite"
)

func DBPath() string {
	if dbPath := os.Getenv("GNOME_AWAIR_CLIENT_DB_PATH"); dbPath != "" {
		return dbPath
	}

	return filepath.Join(DataDir(), DB_NAME)
}
