package app

import (
	"fmt"

	adw "github.com/diamondburned/gotk4-adwaita/pkg/adw"
	gio "github.com/diamondburned/gotk4/pkg/gio/v2"
	gtk "github.com/diamondburned/gotk4/pkg/gtk/v4"
	api "github.com/monorkin/awair-gnome-client/awair/api"
)

type App struct {
	*gtk.Application
	mainWindow *adw.ApplicationWindow
	apiClient  *api.Client
}

func NewApp() *App {
	application := gtk.NewApplication(
		"io.stanko.awair-gnome-client",
		gio.ApplicationFlagsNone,
	)

	app := &App{
		Application: application,
		apiClient:   api.NewClient(),
	}

	app.ConnectActivate(app.onActivate)

	return app
}

func (app *App) onActivate() {
	// Create main application window
	app.mainWindow = adw.NewApplicationWindow(app.Application)
	app.mainWindow.SetDefaultSize(800, 600)
	app.mainWindow.SetTitle("Awair Client")

	// Create the main layout container
	mainBox := gtk.NewBox(gtk.OrientationVertical, 0)

	// Add a header bar with Adwaita styling
	headerBar := adw.NewHeaderBar()
	mainBox.Append(headerBar)

	// Create content area
	contentArea := gtk.NewBox(gtk.OrientationVertical, 12)
	contentArea.SetMarginTop(24)
	contentArea.SetMarginBottom(24)
	contentArea.SetMarginStart(24)
	contentArea.SetMarginEnd(24)
	mainBox.Append(contentArea)

	// Add a title at the top
	titleLabel := gtk.NewLabel("Awair Devices")
	titleLabel.SetHAlign(gtk.AlignStart)
	titleLabel.AddCSSClass("title-1")
	contentArea.Append(titleLabel)

	// Create device list
	devicesBox := app.createDevicesList()
	contentArea.Append(devicesBox)

	// Create bottom action area with refresh button
	actionArea := gtk.NewBox(gtk.OrientationHorizontal, 8)
	actionArea.SetHAlign(gtk.AlignEnd)
	actionArea.SetMarginTop(12)

	refreshButton := gtk.NewButtonWithLabel("Refresh Data")
	refreshButton.ConnectClicked(func() {
		// This would be where you refresh the device data
		// You could call a function like app.refreshDeviceData()
	})
	actionArea.Append(refreshButton)

	contentArea.Append(actionArea)

	// Set the content and show the window
	app.mainWindow.SetContent(mainBox)
	app.mainWindow.Present()

	fmt.Println("Starting device discovery...")
	app.apiClient.StartDeviceDiscovery()
}

func (app *App) createDevicesList() *gtk.Box {
	// Container for device list
	devicesBox := gtk.NewBox(gtk.OrientationVertical, 16)
	devicesBox.SetMarginTop(12)

	// Sample data - in a real app, you'd get this from your Awair devices
	devices := []struct {
		name     string
		temp     float64
		humidity float64
		co2      int
		voc      int
		pm25     int
	}{
		{"Living Room", 22.5, 45.2, 612, 124, 8},
		{"Bedroom", 21.8, 48.7, 580, 89, 5},
		{"Office", 23.1, 42.3, 750, 156, 12},
	}

	// Create cards for each device
	for _, device := range devices {
		// Create a card for the device
		deviceCard := adw.NewPreferencesGroup()
		deviceCard.SetTitle(device.name)
		deviceCard.AddCSSClass("card")

		// Create a grid for the metrics
		grid := gtk.NewGrid()
		grid.SetColumnSpacing(24)
		grid.SetRowSpacing(12)
		grid.SetMarginTop(12)
		grid.SetMarginBottom(12)
		grid.SetMarginStart(12)
		grid.SetMarginEnd(12)

		// Add metrics to the grid
		app.addMetricToGrid(grid, 0, "Temperature", device.temp, "°C")
		app.addMetricToGrid(grid, 1, "Humidity", device.humidity, "%")
		app.addMetricToGrid(grid, 2, "CO₂", float64(device.co2), "ppm")
		app.addMetricToGrid(grid, 3, "VOC", float64(device.voc), "ppb")
		app.addMetricToGrid(grid, 4, "PM2.5", float64(device.pm25), "μg/m³")

		deviceCard.Add(grid)
		devicesBox.Append(deviceCard)
	}

	return devicesBox
}

func (app *App) addMetricToGrid(grid *gtk.Grid, column int, name string, value float64, unit string) {
	// Create label for metric name
	nameLabel := gtk.NewLabel(name)
	nameLabel.SetHAlign(gtk.AlignStart)
	nameLabel.SetXAlign(0)
	grid.Attach(nameLabel, column, 0, 1, 1)

	// Create label for metric value with formatted text
	valueText := formatValue(value, unit)
	valueLabel := gtk.NewLabel("")
	valueLabel.SetMarkup(fmt.Sprintf("<span weight=\"bold\" size=\"larger\">%s</span>", valueText))
	valueLabel.SetHAlign(gtk.AlignStart)
	valueLabel.SetXAlign(0)
	grid.Attach(valueLabel, column, 1, 1, 1)
}

func formatValue(value float64, unit string) string {
	if value == float64(int(value)) {
		return fmt.Sprintf("%d %s", int(value), unit)
	}
	return fmt.Sprintf("%.1f %s", value, unit)
}

func (app *App) Run() int {
	return app.Application.Run(nil)
}
