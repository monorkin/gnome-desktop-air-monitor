package cli

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/app"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/config"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/database"
)

var (
	verbose bool
	logger  *slog.Logger
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gnome-desktop-air-monitor",
	Short: "GNOME Desktop Air Quality Monitor",
	Long: `A GNOME desktop application for monitoring air quality data from Awair devices.
	
The application discovers Awair devices on your network, collects air quality measurements,
and displays them in a user-friendly interface. It also provides a GNOME shell extension
indicator for quick access to air quality information.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default behavior: start the GUI application
		setupLogger()
		initializeApp()
		
		app := app.NewApp()
		os.Exit(app.Run())
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose (debug) logging")
}

// setupLogger configures the logger based on the verbose flag
func setupLogger() {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	
	// Set as default logger
	slog.SetDefault(logger)
}

// initializeApp initializes the database and settings for CLI commands
func initializeApp() {
	// Load settings
	newSettings, settingsLoaded := config.LoadOrInitializeSettingsFromDefaultLocation()
	if newSettings {
		settingsLoaded.Save()
	}
	
	// Initialize database
	database.Init()
}