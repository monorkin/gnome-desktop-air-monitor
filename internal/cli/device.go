package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/monorkin/gnome-desktop-air-monitor/internal/database"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/globals"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/models"
	"github.com/spf13/cobra"
)

// deviceCmd represents the device command
var deviceCmd = &cobra.Command{
	Use:     "device",
	Aliases: []string{"d", "devices"},
	Short:   "Manage and list devices",
	Long:    `Commands for managing and listing discovered air quality monitoring devices.`,
}

// deviceListCmd represents the device list command
var deviceListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all known devices",
	Long:    `List all discovered devices with their ID, name, serial number, IP address, and last seen timestamp.`,
	Run:     runDeviceList,
}

func runDeviceList(cmd *cobra.Command, args []string) {
	globals.Logger.Debug("Fetching devices from database")

	var devices []models.Device
	err := database.DB.Find(&devices).Error
	if err != nil {
		globals.Logger.Error("Failed to fetch devices", "error", err)
		fmt.Fprintf(os.Stderr, "Error: Failed to fetch devices: %v\n", err)
		os.Exit(1)
	}

	if len(devices) == 0 {
		fmt.Println("No devices found.")
		return
	}

	// Create tabwriter for aligned output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Print header
	fmt.Fprintln(w, "ID\tNAME\tSERIAL\tIP ADDRESS\tLAST SEEN")
	fmt.Fprintln(w, "--\t----\t------\t----------\t---------")

	// Print devices
	for _, device := range devices {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
			device.ID,
			device.Name,
			device.SerialNumber,
			device.IPAddress,
			device.LastSeen.Format("2006-01-02T15:04:05Z07:00"),
		)
	}

	globals.Logger.Debug("Device list completed", "count", len(devices))
}

func init() {
	// Add device command to root
	rootCmd.AddCommand(deviceCmd)

	// Add list subcommand to device
	deviceCmd.AddCommand(deviceListCmd)
}

