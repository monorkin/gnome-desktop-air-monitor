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
	"github.com/monorkin/gnome-desktop-air-monitor/internal/config"
	database "github.com/monorkin/gnome-desktop-air-monitor/internal/database"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/models"
)

const (
	APP_IDENTIFIER = "io.stanko.gnome-desktop-air-monitor"
)

var settings *config.Settings

type App struct {
	*gtk.Application
	mainWindow          *adw.ApplicationWindow
	apiClient           *api.Client
	stack               *gtk.Stack
	headerBar           *adw.HeaderBar
	backButton          *gtk.Button
	settingsButton      *gtk.Button
	dbusService         *DBusService
	logger              *slog.Logger
	indexListBox        *gtk.ListBox
	currentDeviceSerial string // Serial number of currently shown device, empty if none
	isEditingDeviceName bool   // Flag to prevent UI refresh while editing device name
}

type DeviceWithMeasurement struct {
	Device      models.Device
	Measurement models.Measurement
}

func NewApp() *App {
	newSettings, settingsLoaded := config.LoadOrInitializeSettingsFromDefaultLocation()
	settings = settingsLoaded
	if newSettings {
		settings.Save()
	}

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
	// Database is already initialized in NewApp()

	// Initialize DBUS service
	var err error
	app.dbusService, err = NewDBusService(app)
	if err != nil {
		app.logger.Error("Failed to initialize DBUS service", "error", err)
	} else {
		app.logger.Info("DBUS service started")
		// Start periodic updates
		app.dbusService.StartPeriodicUpdates()
		// Send initial visibility state
		app.dbusService.EmitVisibilityChanged()
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

// refreshDevicesFromDatabase reloads devices and refreshes the UI
func (app *App) refreshDevicesFromDatabase() {
	// Don't refresh if user is editing device name
	if app.isEditingDeviceName {
		app.logger.Debug("Skipping UI refresh - device name editing in progress")
		return
	}

	app.logger.Debug("Starting UI refresh from database")

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
		return
	}

	// Check if this measurement is for the device shown in shell extension
	app.updateShellExtensionIfNeeded(dbDevice.SerialNumber)
}

// stopAllDevicePolling stops polling for all devices
func (app *App) stopAllDevicePolling() {
	app.logger.Info("Stopping all device polling")
	devices := app.apiClient.GetDevices()
	for _, device := range devices {
		device.StopPolling()
	}
}

// getDevicesWithMeasurements loads all devices with their latest measurements from the database
func (app *App) getDevicesWithMeasurements() ([]DeviceWithMeasurement, error) {
	var devices []models.Device
	err := database.DB.Find(&devices).Error
	if err != nil {
		return nil, err
	}

	devicesWithMeasurements := make([]DeviceWithMeasurement, 0, len(devices))

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
		}

		devicesWithMeasurements = append(devicesWithMeasurements, deviceWithMeasurement)
	}

	return devicesWithMeasurements, nil
}

// getSelectedDeviceForShellExtension returns the device that should be displayed in the shell extension
func (app *App) getSelectedDeviceForShellExtension() (*DeviceWithMeasurement, error) {
	devices, err := app.getDevicesWithMeasurements()
	if err != nil {
		return nil, err
	}

	if len(devices) == 0 {
		return nil, nil
	}

	// If a device is configured in settings, use that one
	if settings.StatusBarDeviceSerialNumber != nil {
		for i := range devices {
			if devices[i].Device.SerialNumber == *settings.StatusBarDeviceSerialNumber {
				return &devices[i], nil
			}
		}
		app.logger.Debug("Configured device not found, falling back to first device", "configured_serial", *settings.StatusBarDeviceSerialNumber)
	}

	// Fall back to first device
	return &devices[0], nil
}

// updateShellExtensionIfNeeded updates the shell extension if the measurement is for the selected device
func (app *App) updateShellExtensionIfNeeded(deviceSerial string) {
	selectedDevice, err := app.getSelectedDeviceForShellExtension()
	if err != nil {
		app.logger.Error("Failed to get selected device for shell extension", "error", err)
		return
	}

	if selectedDevice == nil {
		app.logger.Debug("No device available for shell extension")
		return
	}

	// Check if this measurement is for the selected device
	if selectedDevice.Device.SerialNumber == deviceSerial {
		app.logger.Debug("Updating shell extension with new measurement", "device_serial", deviceSerial)

		// Trigger DBus signal to update shell extension
		if app.dbusService != nil {
			app.dbusService.EmitDeviceUpdated()
		}
	}
}

// setupDeviceDropdown creates and configures the device selection dropdown
func (app *App) setupDeviceDropdown(deviceRow *adw.ActionRow) {
	// Create string list model for the dropdown
	stringList := gtk.NewStringList(nil)
	
	// Create dropdown
	dropdown := gtk.NewDropDown(stringList, nil)
	dropdown.SetHExpand(false)
	dropdown.SetVAlign(gtk.AlignCenter)
	
	// Add dropdown to the row
	deviceRow.AddSuffix(dropdown)
	
	// Load devices and populate dropdown
	app.refreshDeviceDropdown(dropdown, stringList)
	
	// Connect to selection changes
	dropdown.Connect("notify::selected", func() {
		selectedIndex := dropdown.Selected()
		app.onDeviceSelectionChanged(uint32(selectedIndex), stringList)
	})
}

// refreshDeviceDropdown refreshes the device dropdown with current devices
func (app *App) refreshDeviceDropdown(dropdown *gtk.DropDown, stringList *gtk.StringList) {
	// Clear existing items
	stringList.Splice(0, stringList.NItems(), nil)
	
	// Add "No device selected" option
	stringList.Append("No device selected")
	
	// Load devices from database
	devices, err := app.getDevicesWithMeasurements()
	if err != nil {
		app.logger.Error("Failed to load devices for dropdown", "error", err)
		return
	}
	
	// Add devices to dropdown
	selectedIndex := uint32(0) // Default to "No device selected"
	for i, deviceData := range devices {
		displayName := deviceData.Device.Name
		if displayName == "" {
			displayName = deviceData.Device.SerialNumber
		}
		stringList.Append(displayName)
		
		// Check if this device is currently selected in settings
		if settings.StatusBarDeviceSerialNumber != nil && 
		   deviceData.Device.SerialNumber == *settings.StatusBarDeviceSerialNumber {
			selectedIndex = uint32(i + 1) // +1 because of "No device selected" option
		}
	}
	
	// Set the current selection
	dropdown.SetSelected(uint(selectedIndex))
}

// onDeviceSelectionChanged handles device selection changes in the dropdown
func (app *App) onDeviceSelectionChanged(selectedIndex uint32, stringList *gtk.StringList) {
	if selectedIndex == 0 {
		// "No device selected" option chosen
		settings.StatusBarDeviceSerialNumber = nil
	} else {
		// Get devices to find the selected one
		devices, err := app.getDevicesWithMeasurements()
		if err != nil {
			app.logger.Error("Failed to get devices for selection", "error", err)
			return
		}
		
		deviceIndex := int(selectedIndex - 1) // -1 because of "No device selected" option
		if deviceIndex >= 0 && deviceIndex < len(devices) {
			selectedSerial := devices[deviceIndex].Device.SerialNumber
			settings.StatusBarDeviceSerialNumber = &selectedSerial
			app.logger.Info("Device selected for status bar", "device_serial", selectedSerial)
		}
	}
	
	// Save settings
	err := settings.Save()
	if err != nil {
		app.logger.Error("Failed to save settings", "error", err)
		return
	}
	
	// Update shell extension with new selection
	if app.dbusService != nil {
		app.dbusService.EmitDeviceUpdated()
	}
}

// setupVisibilityToggle configures the shell extension visibility toggle
func (app *App) setupVisibilityToggle(visibilitySwitch *gtk.Switch) {
	// Set initial state based on settings
	visibilitySwitch.SetActive(settings.ShowShellExtension)
	
	// Connect to state changes
	visibilitySwitch.Connect("state-set", func(state bool) bool {
		app.onVisibilityToggleChanged(state)
		return false // Allow the state change to proceed
	})
}

// onVisibilityToggleChanged handles changes to the shell extension visibility setting
func (app *App) onVisibilityToggleChanged(visible bool) {
	app.logger.Info("Shell extension visibility changed", "visible", visible)
	
	// Update settings
	settings.ShowShellExtension = visible
	
	// Save settings
	err := settings.Save()
	if err != nil {
		app.logger.Error("Failed to save visibility setting", "error", err)
		return
	}
	
	// Update shell extension
	if app.dbusService != nil {
		app.dbusService.EmitVisibilityChanged()
	}
}
