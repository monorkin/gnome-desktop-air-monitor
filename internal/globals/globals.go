package globals

import (
	"log/slog"
	"os"
	"sync"

	"github.com/monorkin/gnome-desktop-air-monitor/internal/config"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/database"
)

var (
	// Global instances
	Settings *config.Settings
	Logger   *slog.Logger

	// Ensure initialization happens only once
	initOnce sync.Once
)

// Initialize sets up global instances exactly once
func Initialize(verbose bool) {
	initOnce.Do(func() {
		// Setup logger first
		setupLogger(verbose)
		
		Logger.Debug("Initializing global instances")
		
		// Load or create settings
		newSettings, settingsLoaded := config.LoadOrInitializeSettingsFromDefaultLocation()
		Settings = settingsLoaded
		if newSettings {
			Logger.Debug("Created new settings file")
			if err := Settings.Save(); err != nil {
				Logger.Error("Failed to save new settings", "error", err)
			}
		} else {
			Logger.Debug("Loaded existing settings")
		}
		
		// Initialize database
		database.Init()
		Logger.Debug("Database initialized")
		
		Logger.Info("Global initialization completed", "verbose", verbose)
	})
}

// setupLogger configures the global logger
func setupLogger(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	
	Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	
	// Set as default logger
	slog.SetDefault(Logger)
}

// MustBeInitialized panics if globals haven't been initialized
func MustBeInitialized() {
	if Settings == nil || Logger == nil {
		panic("globals not initialized - call globals.Initialize() first")
	}
}