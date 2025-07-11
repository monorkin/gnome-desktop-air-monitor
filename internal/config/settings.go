package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Settings struct {
	StatusBarDeviceSerialNumber *string `json:"status_bar_device_serial_number"`
	DataRetentionPeriod         int     `json:"data_retention_period,omitempty"` // in days, optional
	ShowShellExtension          bool    `json:"show_shell_extension"`
}

func DefaultSettingsPath() string {
	return filepath.Join(ConfigDir(), "settings.json")
}

func LoadOrInitializeSettingsFromDefaultLocation() (bool, *Settings) {
	return LoadOrInitializeSettings(DefaultSettingsPath())
}

func LoadOrInitializeSettings(path string) (bool, *Settings) {
	if settings, err := LoadSettings(path); err == nil {
		return false, settings
	}

	return true, &Settings{
		StatusBarDeviceSerialNumber: nil,
		DataRetentionPeriod:         7,
		ShowShellExtension:          true,
	}
}

func LoadSettings(path string) (*Settings, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

func (s *Settings) Save() error {
	return s.SaveTo(DefaultSettingsPath())
}

func (s *Settings) SaveTo(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
