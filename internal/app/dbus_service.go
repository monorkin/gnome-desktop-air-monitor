package app

import (
	"fmt"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/globals"
)

const (
	dbusName      = "io.stanko.AirMonitor"
	dbusPath      = "/io/stanko/AirMonitor"
	dbusInterface = "io.stanko.AirMonitor"
)

// DBusService handles DBUS communication for the air monitor
type DBusService struct {
	app  *App
	conn *dbus.Conn
}

// NewDBusService creates a new DBUS service for the app
func NewDBusService(app *App) (*DBusService, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to session bus: %w", err)
	}

	service := &DBusService{
		app:  app,
		conn: conn,
	}

	// Export the service object
	err = conn.Export(service, dbus.ObjectPath(dbusPath), dbusInterface)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to export service: %w", err)
	}

	// Export introspection data
	node := &introspect.Node{
		Name: dbusPath,
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			{
				Name: dbusInterface,
				Methods: []introspect.Method{
					{
						Name: "GetSelectedDevice",
						Args: []introspect.Arg{
							{Name: "device", Direction: "out", Type: "a{sv}"},
						},
					},
					{
						Name: "OpenApp",
					},
					{
						Name: "OpenSettings",
					},
					{
						Name: "Quit",
					},
					{
						Name: "GetVisibility",
						Args: []introspect.Arg{
							{Name: "visible", Direction: "out", Type: "b"},
						},
					},
				},
				Signals: []introspect.Signal{
					{
						Name: "DeviceUpdated",
						Args: []introspect.Arg{
							{Name: "device", Type: "a{sv}"},
						},
					},
					{
						Name: "VisibilityChanged",
						Args: []introspect.Arg{
							{Name: "visible", Type: "b"},
						},
					},
				},
			},
		},
	}

	err = conn.Export(introspect.NewIntrospectable(node), dbus.ObjectPath(dbusPath), "org.freedesktop.DBus.Introspectable")
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to export introspection: %w", err)
	}

	// Request the bus name
	reply, err := conn.RequestName(dbusName, dbus.NameFlagDoNotQueue)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to request bus name: %w", err)
	}

	if reply != dbus.RequestNameReplyPrimaryOwner {
		conn.Close()
		return nil, fmt.Errorf("name already taken")
	}

	return service, nil
}

// GetSelectedDevice returns the currently selected device data
func (s *DBusService) GetSelectedDevice() (map[string]dbus.Variant, *dbus.Error) {
	// Get the selected device from settings or first device
	selectedDevice, err := s.app.getSelectedDeviceForShellExtension()
	if err != nil {
		s.app.logger.Error("Failed to get selected device for DBus", "error", err)
		return map[string]dbus.Variant{}, nil
	}
	
	if selectedDevice == nil {
		return map[string]dbus.Variant{}, nil
	}

	return map[string]dbus.Variant{
		"name":        dbus.MakeVariant(selectedDevice.Device.Name),
		"score":       dbus.MakeVariant(selectedDevice.Measurement.Score),
		"temperature": dbus.MakeVariant(selectedDevice.Measurement.Temperature),
		"humidity":    dbus.MakeVariant(selectedDevice.Measurement.Humidity),
		"co2":         dbus.MakeVariant(selectedDevice.Measurement.CO2),
		"voc":         dbus.MakeVariant(selectedDevice.Measurement.VOC),
		"pm25":        dbus.MakeVariant(selectedDevice.Measurement.PM25),
		"timestamp":   dbus.MakeVariant(selectedDevice.Measurement.Timestamp.Unix()),
	}, nil
}

// OpenApp shows the main application window
func (s *DBusService) OpenApp() *dbus.Error {
	// Show the main window if it's hidden
	if s.app.mainWindow != nil {
		s.app.mainWindow.Present()
		s.app.indexPage.show(s.app)
	}
	return nil
}

// OpenSettings shows the settings page
func (s *DBusService) OpenSettings() *dbus.Error {
	// Show the main window and navigate to settings
	if s.app.mainWindow != nil {
		s.app.mainWindow.Present()
		s.app.settingsPage.show(s.app)
	}
	return nil
}

// Quit terminates the application
func (s *DBusService) Quit() *dbus.Error {
	// Quit the application
	s.app.Quit()
	return nil
}

// GetVisibility returns the current shell extension visibility setting
func (s *DBusService) GetVisibility() (bool, *dbus.Error) {
	return globals.Settings.ShowShellExtension, nil
}

// EmitDeviceUpdated sends a device update signal
func (s *DBusService) EmitDeviceUpdated() error {
	selectedDevice, err := s.app.getSelectedDeviceForShellExtension()
	if err != nil {
		s.app.logger.Error("Failed to get selected device for signal", "error", err)
		return err
	}
	
	if selectedDevice == nil {
		return nil
	}

	deviceData := map[string]dbus.Variant{
		"name":        dbus.MakeVariant(selectedDevice.Device.Name),
		"score":       dbus.MakeVariant(selectedDevice.Measurement.Score),
		"temperature": dbus.MakeVariant(selectedDevice.Measurement.Temperature),
		"humidity":    dbus.MakeVariant(selectedDevice.Measurement.Humidity),
		"co2":         dbus.MakeVariant(selectedDevice.Measurement.CO2),
		"voc":         dbus.MakeVariant(selectedDevice.Measurement.VOC),
		"pm25":        dbus.MakeVariant(selectedDevice.Measurement.PM25),
		"timestamp":   dbus.MakeVariant(selectedDevice.Measurement.Timestamp.Unix()),
	}

	return s.conn.Emit(dbus.ObjectPath(dbusPath), dbusInterface+".DeviceUpdated", deviceData)
}

// EmitVisibilityChanged sends a visibility change signal
func (s *DBusService) EmitVisibilityChanged() error {
	// Get visibility setting from the app's settings
	visible := globals.Settings.ShowShellExtension
	
	s.app.logger.Debug("Emitting visibility changed signal", "visible", visible)
	
	return s.conn.Emit(dbus.ObjectPath(dbusPath), dbusInterface+".VisibilityChanged", visible)
}

// StartPeriodicUpdates begins sending periodic device updates
func (s *DBusService) StartPeriodicUpdates() {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			if err := s.EmitDeviceUpdated(); err != nil {
				fmt.Printf("Failed to emit device update: %v\n", err)
			}
		}
	}()
}

// Close closes the DBUS connection
func (s *DBusService) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}
