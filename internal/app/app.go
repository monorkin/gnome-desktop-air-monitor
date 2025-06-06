package app

import (
	"fmt"

	adw "github.com/diamondburned/gotk4-adwaita/pkg/adw"
	gio "github.com/diamondburned/gotk4/pkg/gio/v2"
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
	mainWindow     *adw.ApplicationWindow
	apiClient      *api.Client
	stack          *gtk.Stack
	headerBar      *adw.HeaderBar
	backButton     *gtk.Button
	settingsButton *gtk.Button
	devices        []DeviceWithMeasurement
	dbusService    *DBusService
}

type DeviceWithMeasurement struct {
	Device      models.Device
	Measurement models.Measurement
}

func NewApp() *App {
	database.Init()

	application := gtk.NewApplication(
		APP_IDENTIFIER,
		gio.ApplicationFlagsNone,
	)

	app := &App{
		Application: application,
		apiClient:   api.NewClient(),
	}

	app.ConnectActivate(app.onActivate)
	
	// Hold the application so it doesn't quit when the window is closed
	app.Hold()

	return app
}

func (app *App) onActivate() {
	app.generateMockData()

	// Initialize DBUS service
	var err error
	app.dbusService, err = NewDBusService(app)
	if err != nil {
		fmt.Printf("Failed to initialize DBUS service: %v\n", err)
	} else {
		fmt.Println("DBUS service started")
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

	fmt.Println("Starting device discovery...")
	app.apiClient.StartDeviceDiscovery()
}

func (app *App) Run() int {
	return app.Application.Run(nil)
}

func (app *App) Quit() {
	// Close DBUS service
	if app.dbusService != nil {
		app.dbusService.Close()
	}
	
	// Release the hold and quit
	app.Release()
	app.Application.Quit()
}
