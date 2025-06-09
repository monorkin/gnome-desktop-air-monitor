package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/monorkin/gnome-desktop-air-monitor/internal/database"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/globals"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/models"
	"github.com/spf13/cobra"
)

// measurementCmd represents the measurement command
var measurementCmd = &cobra.Command{
	Use:     "measurement",
	Aliases: []string{"m", "measurements"},
	Short:   "Get measurement data",
	Long:    `Commands for retrieving measurement data from devices.`,
}

// measurementGetCmd represents the measurement get command
var measurementGetCmd = &cobra.Command{
	Use:   "get <device_id_or_serial>",
	Short: "Get the latest measurement for a device",
	Long: `Get the latest measurement for a device specified by either device ID or serial number.

Examples:
  gnome-desktop-air-monitor measurement get 1
  gnome-desktop-air-monitor measurement get awair-element_12345`,
	Args: cobra.ExactArgs(1),
	Run:  runMeasurementGet,
}

func runMeasurementGet(cmd *cobra.Command, args []string) {
	deviceIdentifier := args[0]
	globals.Logger.Debug("Getting measurement for device", "identifier", deviceIdentifier)

	// First try to find device by ID, then by serial number
	var device models.Device
	var err error

	// Try parsing as ID first
	if deviceID, parseErr := strconv.ParseUint(deviceIdentifier, 10, 32); parseErr == nil {
		err = database.DB.First(&device, uint(deviceID)).Error
	} else {
		// Try finding by serial number
		err = database.DB.Where("serial_number = ?", deviceIdentifier).First(&device).Error
	}

	if err != nil {
		globals.Logger.Error("Device not found", "identifier", deviceIdentifier, "error", err)
		fmt.Fprintf(os.Stderr, "Error: Device not found: %s\n", deviceIdentifier)
		os.Exit(1)
	}

	globals.Logger.Debug("Found device", "id", device.ID, "name", device.Name, "serial", device.SerialNumber)

	// Get the latest measurement for this device
	var measurement models.Measurement
	err = database.DB.Where("device_id = ?", device.ID).
		Order("timestamp DESC").
		First(&measurement).Error
	if err != nil {
		globals.Logger.Error("No measurements found for device", "device_id", device.ID, "error", err)
		fmt.Fprintf(os.Stderr, "Error: No measurements found for device %s\n", deviceIdentifier)
		os.Exit(1)
	}

	// Create response structure
	response := struct {
		Device      DeviceInfo      `json:"device"`
		Measurement MeasurementInfo `json:"measurement"`
	}{
		Device: DeviceInfo{
			ID:           device.ID,
			Name:         device.Name,
			SerialNumber: device.SerialNumber,
			IPAddress:    device.IPAddress,
			DeviceType:   device.DeviceType,
			LastSeen:     device.LastSeen.Format("2006-01-02T15:04:05Z07:00"),
		},
		Measurement: MeasurementInfo{
			Timestamp:   measurement.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
			Temperature: measurement.Temperature,
			Humidity:    measurement.Humidity,
			CO2:         measurement.CO2,
			VOC:         measurement.VOC,
			PM25:        measurement.PM25,
			Score:       measurement.Score,
		},
	}

	// Output as JSON
	output, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		globals.Logger.Error("Failed to marshal response", "error", err)
		fmt.Fprintf(os.Stderr, "Error: Failed to format response: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))

	globals.Logger.Debug("Measurement get completed", "device_id", device.ID)
}

// DeviceInfo represents device information for JSON output
type DeviceInfo struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	SerialNumber string `json:"serial_number"`
	IPAddress    string `json:"ip_address"`
	DeviceType   string `json:"device_type"`
	LastSeen     string `json:"last_seen"`
}

// MeasurementInfo represents measurement information for JSON output
type MeasurementInfo struct {
	Timestamp   string  `json:"timestamp"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	CO2         float64 `json:"co2"`
	VOC         float64 `json:"voc"`
	PM25        float64 `json:"pm25"`
	Score       float64 `json:"score"`
}

func init() {
	// Add measurement command to root
	rootCmd.AddCommand(measurementCmd)

	// Add get subcommand to measurement
	measurementCmd.AddCommand(measurementGetCmd)
}

