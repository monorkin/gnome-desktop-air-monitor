package app

import (
	"fmt"
	"time"

	"github.com/monorkin/gnome-desktop-air-monitor/internal/models"
)

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
