package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/app"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/globals"
)

var verbose bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gnome-desktop-air-monitor",
	Short: "GNOME Desktop Air Quality Monitor",
	Long: `A GNOME desktop application for monitoring air quality data from Awair devices.
	
The application discovers Awair devices on your network, collects air quality measurements,
and displays them in a user-friendly interface. It also provides a GNOME shell extension
indicator for quick access to air quality information.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize globals before any command runs
		globals.Initialize(verbose)
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Default behavior: start the GUI application
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