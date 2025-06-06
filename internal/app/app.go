package app

import (
	"log/slog"
	"os"
	"time"

	adw "github.com/diamondburned/gotk4-adwaita/pkg/adw"
	gio "github.com/diamondburned/gotk4/pkg/gio/v2"
	glib "github.com/diamondburned/gotk4/pkg/glib/v2"
	gtk "github.com/diamondburned/gotk4/pkg/gtk/v4"
	api "github.com/monorkin/gnome-desktop-air-monitor/awair/api"
	database "github.com/monorkin/gnome-desktop-air-monitor/internal/database"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/models"
)

const (
	APP_IDENTIFIER = "io.stanko.gnome-desktop-air-monitor"
)

type App struct {
	*gtk.Application
	mainWindow           *adw.ApplicationWindow
	apiClient            *api.Client
	stack                *gtk.Stack
	headerBar            *adw.HeaderBar
	backButton           *gtk.Button
	settingsButton       *gtk.Button
	devices              []DeviceWithMeasurement
	dbusService          *DBusService
	logger               *slog.Logger
	indexListBox         *gtk.ListBox
	currentDeviceSerial  string // Serial number of currently shown device, empty if none
}

type DeviceWithMeasurement struct {
	Device      models.Device
	Measurement models.Measurement
}

func NewApp() *App {
	database.Init()

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	application := gtk.NewApplication(
		APP_IDENTIFIER,
		gio.ApplicationFlagsNone,
	)

	app := &App{
		Application: application,
		apiClient:   api.NewClientWithLogger(logger),
		logger:      logger,
	}

	app.ConnectActivate(app.onActivate)

	// Hold the application so it doesn't quit when the window is closed
	app.Hold()

	return app
}

func (app *App) onActivate() {
	// Load devices from database instead of generating mock data
	err := app.loadDevicesFromDatabase()
	if err != nil {
		app.logger.Error("Failed to load devices from database", "error", err)
		// Initialize empty device list
		app.devices = make([]DeviceWithMeasurement, 0)
	}

	// Initialize DBUS service
	app.dbusService, err = NewDBusService(app)
	if err != nil {
		app.logger.Error("Failed to initialize DBUS service", "error", err)
	} else {
		app.logger.Info("DBUS service started")
		// Start periodic updates
		app.dbusService.StartPeriodicUpdates()
	}

	app.mainWindow = adw.NewApplicationWindow(app.Application)
	app.mainWindow.SetDefaultSize(800, 600)
	app.mainWindow.SetTitle("Air Monitor")

	// Make window hide instead of quit when closed
	app.mainWindow.ConnectCloseRequest(func() bool {
		app.mainWindow.SetVisible(false)
		return true // Prevent default close behavior
	})

	mainBox := gtk.NewBox(gtk.OrientationVertical, 0)

	app.headerBar = adw.NewHeaderBar()

	app.backButton = gtk.NewButtonFromIconName("go-previous-symbolic")
	app.backButton.SetVisible(false)
	app.backButton.ConnectClicked(func() {
		app.showIndexPage()
	})
	app.headerBar.PackStart(app.backButton)

	app.settingsButton = gtk.NewButtonFromIconName("preferences-system-symbolic")
	app.settingsButton.ConnectClicked(func() {
		app.showSettingsPage()
	})
	app.headerBar.PackEnd(app.settingsButton)

	mainBox.Append(app.headerBar)

	app.stack = gtk.NewStack()
	app.stack.SetTransitionType(gtk.StackTransitionTypeSlideLeftRight)
	mainBox.Append(app.stack)

	app.setupIndexPage()
	app.setupSettingsPage()
	app.mainWindow.SetContent(mainBox)
	app.mainWindow.Present()

	app.apiClient.SetOnDeviceDiscovered(app.onDeviceDiscovered)
	app.apiClient.StartDeviceDiscovery()
}

func (app *App) Run() int {
	return app.Application.Run(nil)
}

func (app *App) Quit() {
	// Stop device polling
	app.stopAllDevicePolling()

	// Close DBUS service
	if app.dbusService != nil {
		app.dbusService.Close()
	}

	// Release the hold and quit
	app.Release()
	app.Application.Quit()
}

// onDeviceDiscovered is called when a new device is discovered by the API client
func (app *App) onDeviceDiscovered(apiDevice api.Device) {
	app.logger.Info("Device discovered", "hostname", apiDevice.Hostname, "ip", apiDevice.IP)

	// Convert API device to database model
	dbDevice := app.convertAPIDeviceToModel(apiDevice)

	// Store device in database
	err := app.storeDevice(dbDevice)
	if err != nil {
		app.logger.Error("Failed to store device", "hostname", apiDevice.Hostname, "error", err)
		return
	}

	app.logger.Info("Device stored successfully", "name", dbDevice.Name, "serial", dbDevice.SerialNumber)

	// Store initial measurement if available
	if apiDevice.LastMeasurement != nil {
		err := app.storeMeasurement(dbDevice.ID, *apiDevice.LastMeasurement)
		if err != nil {
			app.logger.Error("Failed to store initial measurement", "device_id", dbDevice.ID, "error", err)
		} else {
			app.logger.Debug("Initial measurement stored", "device_id", dbDevice.ID)
		}
	}

	// Start polling for measurements
	app.startDevicePolling(apiDevice)
}

// convertAPIDeviceToModel converts an API device to a database model
func (app *App) convertAPIDeviceToModel(apiDevice api.Device) models.Device {
	var deviceType string
	if apiDevice.Type != nil {
		deviceType = string(*apiDevice.Type)
	} else {
		deviceType = string(api.DeviceTypeUnknown)
	}

	var serialNumber string
	if apiDevice.ID != nil {
		serialNumber = *apiDevice.ID
	} else {
		// Fallback to hostname if ID is not available
		serialNumber = apiDevice.Hostname
	}

	return models.Device{
		Name:         apiDevice.Hostname,
		IPAddress:    apiDevice.IP,
		DeviceType:   deviceType,
		SerialNumber: serialNumber,
		LastSeen:     time.Now(),
	}
}

// storeDevice stores a device in the database, updating if it already exists
func (app *App) storeDevice(device models.Device) error {
	// Check if device already exists by serial number
	var existingDevice models.Device
	result := database.DB.Where("serial_number = ?", device.SerialNumber).First(&existingDevice)

	if result.Error == nil {
		// Device exists, update it
		existingDevice.IPAddress = device.IPAddress
		existingDevice.LastSeen = device.LastSeen

		err := database.DB.Save(&existingDevice).Error
		if err == nil {
			// Refresh the UI after storing device
			app.refreshDevicesFromDatabaseSafe()
		}
		return err
	} else {
		// Device doesn't exist, create it
		err := database.DB.Create(&device).Error
		if err == nil {
			// Refresh the UI after storing device
			app.refreshDevicesFromDatabaseSafe()
		}
		return err
	}
}

// loadDevicesFromDatabase loads all devices with their latest measurements from the database
func (app *App) loadDevicesFromDatabase() error {
	var devices []models.Device
	err := database.DB.Find(&devices).Error
	if err != nil {
		app.logger.Error("Failed to load devices from database", "error", err)
		return err
	}

	app.devices = make([]DeviceWithMeasurement, 0, len(devices))

	for _, device := range devices {
		// Get the latest measurement for this device
		var measurement models.Measurement
		err := database.DB.Where("device_id = ?", device.ID).
			Order("timestamp DESC").
			First(&measurement).Error

		deviceWithMeasurement := DeviceWithMeasurement{
			Device: device,
		}

		if err == nil {
			// Found measurement
			deviceWithMeasurement.Measurement = measurement
		} else {
			// No measurement found, create a placeholder
			deviceWithMeasurement.Measurement = models.Measurement{
				DeviceID:    device.ID,
				Timestamp:   time.Now(),
				Temperature: 0,
				Humidity:    0,
				CO2:         0,
				VOC:         0,
				PM25:        0,
				Score:       0,
			}
			app.logger.Debug("No measurements found for device", "device_id", device.ID, "device_name", device.Name)
		}

		app.devices = append(app.devices, deviceWithMeasurement)
	}

	app.logger.Info("Loaded devices from database", "count", len(app.devices))
	return nil
}

// refreshDevicesFromDatabase reloads devices and refreshes the UI
func (app *App) refreshDevicesFromDatabase() {
	app.logger.Debug("Starting UI refresh from database")
	err := app.loadDevicesFromDatabase()
	if err != nil {
		app.logger.Error("Failed to refresh devices from database", "error", err)
		return
	}

	app.logger.Debug("Refreshing UI components", "current_device", app.currentDeviceSerial)

	// Refresh the index page if it exists
	app.refreshIndexPage()
	
	// Refresh the current device page if one is shown
	app.refreshCurrentDevicePage()
	
	app.logger.Debug("UI refresh completed")
}

// refreshDevicesFromDatabaseSafe safely refreshes devices from the main thread
func (app *App) refreshDevicesFromDatabaseSafe() {
	app.logger.Debug("Scheduling UI refresh via glib.IdleAdd")
	glib.IdleAdd(func() bool {
		app.logger.Debug("Executing UI refresh from main thread")
		app.refreshDevicesFromDatabase()
		return false // Don't repeat
	})
}

// storeMeasurement stores a measurement in the database
func (app *App) storeMeasurement(deviceID uint, apiMeasurement api.Measurement) error {
	measurement := models.Measurement{
		DeviceID:    deviceID,
		Timestamp:   apiMeasurement.Timestamp,
		Temperature: apiMeasurement.Temperature,
		Humidity:    apiMeasurement.Humidity,
		CO2:         float64(apiMeasurement.CO2),
		VOC:         float64(apiMeasurement.VOC),
		PM25:        float64(apiMeasurement.PM25),
		Score:       float64(apiMeasurement.Score),
	}

	// If timestamp is zero, use current time
	if measurement.Timestamp.IsZero() {
		measurement.Timestamp = time.Now()
	}

	err := database.DB.Create(&measurement).Error
	if err == nil {
		app.logger.Debug("Measurement stored", "device_id", deviceID, "score", measurement.Score)
		// Refresh UI after storing measurement (safely from any thread)
		app.refreshDevicesFromDatabaseSafe()
	}
	return err
}

// startDevicePolling starts polling for a discovered device
func (app *App) startDevicePolling(apiDevice api.Device) {
	// Find the device in the client's device map to start polling
	devices := app.apiClient.GetDevices()
	for _, device := range devices {
		if device.ID != nil && apiDevice.ID != nil && *device.ID == *apiDevice.ID {
			// Set up callback for new measurements
			device.SetOnMeasurement(func(measurement *api.Measurement) {
				app.onDeviceMeasurement(apiDevice, measurement)
			})
			
			// Start polling
			device.StartPolling()
			app.logger.Info("Started polling for device", "device_id", *device.ID, "hostname", device.Hostname)
			break
		}
	}
}

// onDeviceMeasurement is called when a new measurement is received from polling
func (app *App) onDeviceMeasurement(apiDevice api.Device, measurement *api.Measurement) {
	app.logger.Debug("New measurement received", "device_id", *apiDevice.ID, "score", measurement.Score)

	// Find the device in database to get its ID
	var dbDevice models.Device
	err := database.DB.Where("serial_number = ?", *apiDevice.ID).First(&dbDevice).Error
	if err != nil {
		app.logger.Error("Failed to find device for measurement", "device_id", *apiDevice.ID, "error", err)
		return
	}

	// Store the measurement
	err = app.storeMeasurement(dbDevice.ID, *measurement)
	if err != nil {
		app.logger.Error("Failed to store measurement", "device_id", dbDevice.ID, "error", err)
	}
}

// stopAllDevicePolling stops polling for all devices
func (app *App) stopAllDevicePolling() {
	app.logger.Info("Stopping all device polling")
	devices := app.apiClient.GetDevices()
	for _, device := range devices {
		device.StopPolling()
	}
}
