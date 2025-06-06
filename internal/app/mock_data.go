package app

import (
	"fmt"
	"time"

	database "github.com/monorkin/gnome-desktop-air-monitor/internal/database"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/models"
)

// generateMockData creates mock device data for testing purposes
// This is now unused since we load data from the database
func (app *App) generateMockData() {
	rooms := []string{"Living Room", "Bedroom", "Office", "Kitchen", "Bathroom", "Guest Room", "Study", "Basement", "Attic", "Garage", "Dining Room", "Nursery"}

	app.devices = make([]DeviceWithMeasurement, len(rooms))

	for i, room := range rooms {
		app.devices[i] = DeviceWithMeasurement{
			Device: models.Device{
				Name:         fmt.Sprintf("Awair Element-%d", i+1),
				IPAddress:    fmt.Sprintf("192.168.1.%d", 100+i),
				DeviceType:   "Element",
				SerialNumber: fmt.Sprintf("AWR%d%04d", 2023, 1000+i),
				LastSeen:     time.Now().Add(-time.Duration(i*5) * time.Minute),
			},
			Measurement: models.Measurement{
				Timestamp:   time.Now().Add(-time.Duration(i*2) * time.Minute),
				Temperature: 20.0 + float64(i%8),
				Humidity:    40.0 + float64(i*3%20),
				CO2:         400.0 + float64(i*50),
				VOC:         50.0 + float64(i*10),
				PM25:        5.0 + float64(i%15),
				Score:       float64(25 + i*6%70),
			},
		}
		app.devices[i].Device.Name = room
	}
}

// generateMockDataInDatabase creates mock device data directly in the database for testing
func (app *App) generateMockDataInDatabase() error {
	rooms := []string{"Living Room", "Bedroom", "Office"}

	for i, room := range rooms {
		device := models.Device{
			Name:         room,
			IPAddress:    fmt.Sprintf("192.168.1.%d", 100+i),
			DeviceType:   "awair-element",
			SerialNumber: fmt.Sprintf("AWR%d%04d", 2023, 1000+i),
			LastSeen:     time.Now().Add(-time.Duration(i*5) * time.Minute),
		}

		err := database.DB.Create(&device).Error
		if err != nil {
			app.logger.Error("Failed to create mock device", "device", room, "error", err)
			continue
		}

		measurement := models.Measurement{
			DeviceID:    device.ID,
			Timestamp:   time.Now().Add(-time.Duration(i*2) * time.Minute),
			Temperature: 20.0 + float64(i%8),
			Humidity:    40.0 + float64(i*3%20),
			CO2:         400.0 + float64(i*50),
			VOC:         50.0 + float64(i*10),
			PM25:        5.0 + float64(i%15),
			Score:       float64(25 + i*6%70),
		}

		err = database.DB.Create(&measurement).Error
		if err != nil {
			app.logger.Error("Failed to create mock measurement", "device", room, "error", err)
		}
	}

	app.logger.Info("Generated mock data in database", "devices", len(rooms))
	return nil
}
