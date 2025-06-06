package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
)

const (
	AWAIR_HOSTNAME_PREFIX = "awair-"
	USER_AGENT            = "Gnome Awair Client/1.0.0"
	REQUEST_TIMEOUT       = 5 * time.Second
	DISCOVERY_INTERVAL    = 20 * time.Second
	DISCOVERY_TIMEOUT     = 10 * time.Second
)

type Client struct {
	httpClient             http.Client
	deviceDiscoveryContext context.Context
	deviceDiscoveryCancel  context.CancelFunc
	devices                map[string]*Device
	devicesMutex           sync.RWMutex
	onDeviceDiscovered     func(Device)
	logger                 *slog.Logger
}

type DeviceInfo struct {
	Type            DeviceType
	ID              string
	FirmwareVersion string
}

func NewClient() *Client {
	return NewClientWithLogger(nil)
}

func NewClientWithLogger(logger *slog.Logger) *Client {
	return &Client{
		httpClient: http.Client{
			Timeout: REQUEST_TIMEOUT,
		},
		devices: make(map[string]*Device),
		logger:  logger,
	}
}

func (client *Client) SetOnDeviceDiscovered(callback func(Device)) {
	client.onDeviceDiscovered = callback
}

func (client *Client) log(level slog.Level, msg string, args ...any) {
	if client.logger != nil {
		client.logger.Log(context.Background(), level, msg, args...)
	}
}

func (client *Client) StartDeviceDiscovery() {
	client.StopDeviceDiscovery()
	client.log(slog.LevelDebug, "Initializing device discovery")

	client.deviceDiscoveryContext, client.deviceDiscoveryCancel = context.WithCancel(context.Background())

	go func() {
		client.log(slog.LevelInfo, "Starting device discovery")
		devices, err := client.discoverDevices(client.deviceDiscoveryContext)
		if err != nil {
			client.log(slog.LevelError, "Error during initial device discovery", "error", err)
		}

		client.log(slog.LevelDebug, "Initial device discovery completed", "devices_count", len(devices))
		client.updateDevices(devices)

		// Then run periodic discovery
		ticker := time.NewTicker(DISCOVERY_INTERVAL)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				devices, err := client.discoverDevices(client.deviceDiscoveryContext)
				if err != nil {
					client.log(slog.LevelError, "Error during periodic device discovery", "error", err)
					continue
				}
				client.log(slog.LevelDebug, "Periodic device discovery completed", "devices_count", len(devices))
				client.updateDevices(devices)
			case <-client.deviceDiscoveryContext.Done():
				client.log(slog.LevelInfo, "Device discovery stopped")
				return
			}
		}
	}()
}

func (client *Client) StopDeviceDiscovery() {
	if client.deviceDiscoveryCancel != nil {
		client.log(slog.LevelDebug, "Stopping device discovery")
		client.deviceDiscoveryCancel()
		client.deviceDiscoveryCancel = nil
	}
}

func (client *Client) discoverDevices(ctx context.Context) ([]*Device, error) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize resolver: %w", err)
	}

	entries := make(chan *zeroconf.ServiceEntry)
	go func() {
		err = resolver.Browse(ctx, "_http._tcp", "local.", entries)
		if err != nil {
			client.log(slog.LevelError, "Failed to browse for devices", "error", err)
		}
	}()

	var devices []*Device
	timeout := time.After(5 * time.Second)

loop:
	for {
		select {
		case entry := <-entries:
			hostname := entry.HostName

			if !strings.EqualFold(hostname[:len(AWAIR_HOSTNAME_PREFIX)], AWAIR_HOSTNAME_PREFIX) {
				continue
			}

			devices = append(devices, &Device{
				Client:   client,
				IP:       entry.AddrIPv4[0].String(),
				Hostname: hostname,
				ID:       nil,
				Type:     nil,
			})
		case <-timeout:
			break loop
		case <-ctx.Done():
			break loop
		}
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(devices))

	for _, device := range devices {
		wg.Add(1)
		go func(device *Device) {
			defer wg.Done()

			client.log(slog.LevelDebug, "Fetching device info", "ip", device.IP, "hostname", device.Hostname)

			if err := device.FetchInfo(); err != nil {
				client.log(slog.LevelError, "Failed to fetch device info", "ip", device.IP, "error", err)
				errChan <- fmt.Errorf("failed to fetch device info for %s: %w", device.IP, err)
			}

			client.log(slog.LevelDebug, "Device info fetched", "ip", device.IP, "ID", *device.ID, "type", *device.Type)
		}(device)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		return devices, err
	}

	for _, device := range devices {
		client.log(slog.LevelDebug, "Device discovered", "ip", device.IP, "hostname", device.Hostname, "ID", *device.ID, "type", *device.Type)
	}

	return devices, nil
}

func (client *Client) updateDevices(devices []*Device) {
	client.devicesMutex.Lock()
	defer client.devicesMutex.Unlock()

	for _, device := range devices {
		if device.ID == nil {
			continue
		}

		existingDevice, exists := client.devices[*device.ID]

		if !exists {
			client.devices[*device.ID] = device

			if client.onDeviceDiscovered != nil {
				go client.onDeviceDiscovered(*device)
			}
		} else {
			device.LastUpdated = existingDevice.LastUpdated
			client.devices[*device.ID] = device
		}
	}
}

func (client *Client) GetDevices() []*Device {
	client.devicesMutex.RLock()
	defer client.devicesMutex.RUnlock()

	devices := make([]*Device, 0, len(client.devices))

	for _, device := range client.devices {
		devices = append(devices, device)
	}

	return devices
}

func (client *Client) FetchDeviceInfo(ip string) (*DeviceInfo, error) {
	url := fmt.Sprintf("http://%s/settings/config/data", ip)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("User-Agent", USER_AGENT)

	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch device info: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch device info: %s", response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	deviceInfo := &DeviceInfo{}

	deviceUUID, ok := data["device_uuid"].(string)
	if !ok {
		return nil, fmt.Errorf("failed to parse deviceUUID")
	}

	deviceInfo.ID = deviceUUID

	firmwareVersion, ok := data["fw_version"].(string)
	if !ok {
		return nil, fmt.Errorf("failed to parse firmware version")
	}

	deviceInfo.FirmwareVersion = firmwareVersion

	lowercaseID := strings.ToLower(deviceInfo.ID)

	switch {
	case strings.HasPrefix(lowercaseID, "awair-element_"):
		deviceInfo.Type = DeviceTypeAwairElement
	case strings.HasPrefix(lowercaseID, "awair-omni_"):
		deviceInfo.Type = DeviceTypeAwairOmni
	default:
		deviceInfo.Type = DeviceTypeUnknown
	}

	return deviceInfo, nil
}

func (client *Client) FetchMeasurment(ip string) (*Measurement, error) {
	url := fmt.Sprintf("http://%s/air-data/latest", ip)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("User-Agent", USER_AGENT)

	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch device info: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch device info: %s", response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var data *Measurement
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	client.log(slog.LevelDebug, "Measurement data fetched", "data", data)

	return data, nil
}
