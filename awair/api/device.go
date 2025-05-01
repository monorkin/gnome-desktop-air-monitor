package api

import (
	"time"
)

type DeviceType string

const (
	DeviceTypeAwairElement DeviceType = "awair-element"
	DeviceTypeAwairOmni    DeviceType = "awair-omni"
	DeviceTypeUnknown      DeviceType = "unknown"
)

type Device struct {
	Client          *Client
	Type            *DeviceType
	ID              *string
	FirmwareVersion *string
	IP              string
	Hostname        string
	LastUpdated     time.Time
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

	device.FetchMeasurement()

	return nil
}

func (device *Device) FetchMeasurement() (*Measurement, error) {
	measurement, err := device.Client.FetchMeasurment(device.IP)

	return measurement, err
}
