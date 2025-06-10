package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	APP_DIR_NAME = "gnome-desktop-air-monitor"
)

func DataDir() string {
	var baseDir string

	if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
		baseDir = filepath.Join(xdgDataHome, APP_DIR_NAME)
	} else {
		homeDir, err := os.UserHomeDir()
		// In case the home directory cannot be determined use the current working directory
		if err != nil {
			currentDir, err := os.Getwd()
			if err != nil {
				return "."
			}

			return currentDir
		}

		localSharePath := filepath.Join(homeDir, ".local", "share")

		if _, err := os.Stat(localSharePath); err == nil {
			baseDir = filepath.Join(localSharePath, APP_DIR_NAME)
		} else {
			baseDir = filepath.Join(homeDir, fmt.Sprintf(".%s", APP_DIR_NAME))
		}
	}

	return baseDir
}

func ConfigDir() string {
	var baseDir string

	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		baseDir = filepath.Join(xdgConfigHome, APP_DIR_NAME)
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "."
		}

		localConfigPath := filepath.Join(homeDir, ".config")

		if _, err := os.Stat(localConfigPath); err == nil {
			baseDir = filepath.Join(localConfigPath, APP_DIR_NAME)
		} else {
			baseDir = filepath.Join(homeDir, fmt.Sprintf(".%s", APP_DIR_NAME))
		}
	}

	return baseDir
}
