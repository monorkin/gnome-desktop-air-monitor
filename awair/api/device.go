package api

import (
	"context"
	"log/slog"
	"time"
)

type DeviceType string

const (
	DeviceTypeAwairElement DeviceType = "awair-element"
	DeviceTypeAwairOmni    DeviceType = "awair-omni"
	DeviceTypeUnknown      DeviceType = "unknown"
)

type Device struct {
	Client             *Client
	Type               *DeviceType
	ID                 *string
	FirmwareVersion    *string
	LastMeasurement    *Measurement
	IP                 string
	Hostname           string
	LastUpdated        time.Time
	pollingContext     context.Context
	pollingCancel      context.CancelFunc
	onMeasurement      func(*Measurement)
}

func (device *Device) FetchInfo() error {
	deviceInfo, err := device.Client.FetchDeviceInfo(device.IP)
	if err != nil {
		return err
	}

	device.ID = &deviceInfo.ID
	device.FirmwareVersion = &deviceInfo.FirmwareVersion
	device.Type = &deviceInfo.Type
	device.LastUpdated = time.Now()

	measurement, err := device.FetchMeasurement()
	if err != nil {
		device.LastMeasurement = measurement
	}

	return nil
}

func (device *Device) FetchMeasurement() (*Measurement, error) {
	measurement, err := device.Client.FetchMeasurment(device.IP)
	if err == nil {
		device.LastMeasurement = measurement
	}

	return measurement, err
}

// SetOnMeasurement sets the callback function for new measurements
func (device *Device) SetOnMeasurement(callback func(*Measurement)) {
	device.onMeasurement = callback
}

// StartPolling starts polling for measurements every 10 seconds
func (device *Device) StartPolling() {
	device.StopPolling()

	if device.Client != nil {
		device.Client.log(slog.LevelInfo, "Starting measurement polling", "device_id", device.ID, "ip", device.IP)
	}

	device.pollingContext, device.pollingCancel = context.WithCancel(context.Background())

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				measurement, err := device.FetchMeasurement()
				if err != nil {
					if device.Client != nil {
						device.Client.log(slog.LevelError, "Failed to fetch measurement", "device_id", device.ID, "ip", device.IP, "error", err)
					}
					continue
				}

				if device.Client != nil {
					device.Client.log(slog.LevelDebug, "Measurement fetched", "device_id", device.ID, "score", measurement.Score, "temp", measurement.Temperature)
				}

				// Trigger callback if set
				if device.onMeasurement != nil {
					go device.onMeasurement(measurement)
				}

			case <-device.pollingContext.Done():
				if device.Client != nil {
					device.Client.log(slog.LevelInfo, "Measurement polling stopped", "device_id", device.ID)
				}
				return
			}
		}
	}()
}

// StopPolling stops the measurement polling
func (device *Device) StopPolling() {
	if device.pollingCancel != nil {
		if device.Client != nil {
			device.Client.log(slog.LevelDebug, "Stopping measurement polling", "device_id", device.ID)
		}
		device.pollingCancel()
		device.pollingCancel = nil
	}
}
