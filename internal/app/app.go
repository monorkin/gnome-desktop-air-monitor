package app

import (
	"fmt"
	"math"
	"time"

	adw "github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/cairo"
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

	return app
}

func (app *App) onActivate() {
	app.generateMockData()

	app.mainWindow = adw.NewApplicationWindow(app.Application)
	app.mainWindow.SetDefaultSize(800, 600)
	app.mainWindow.SetTitle("Air Monitor")

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

func (app *App) setupIndexPage() {
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolled.SetVExpand(true)

	listBox := gtk.NewListBox()
	listBox.SetSelectionMode(gtk.SelectionNone)
	listBox.AddCSSClass("boxed-list")
	listBox.SetMarginTop(24)
	listBox.SetMarginBottom(24)
	listBox.SetMarginStart(24)
	listBox.SetMarginEnd(24)

	for i, deviceData := range app.devices {
		row := app.createDeviceRow(deviceData, i)
		listBox.Append(row)
	}

	scrolled.SetChild(listBox)
	app.stack.AddNamed(scrolled, "index")
	app.stack.SetVisibleChildName("index")
}

func (app *App) createDeviceRow(deviceData DeviceWithMeasurement, index int) *gtk.ListBoxRow {
	row := gtk.NewListBoxRow()
	row.SetActivatable(true)
	
	mainBox := gtk.NewBox(gtk.OrientationHorizontal, 16)
	mainBox.SetMarginTop(12)
	mainBox.SetMarginBottom(12)
	mainBox.SetMarginStart(16)
	mainBox.SetMarginEnd(16)

	scoreCircle := app.createScoreCircle(deviceData.Measurement.Score)
	scoreCircle.SetVAlign(gtk.AlignCenter)
	mainBox.Append(scoreCircle)

	textBox := gtk.NewBox(gtk.OrientationVertical, 4)
	textBox.SetVAlign(gtk.AlignCenter)

	deviceNameLabel := gtk.NewLabel(deviceData.Device.Name)
	deviceNameLabel.SetHAlign(gtk.AlignStart)
	deviceNameLabel.SetXAlign(0)
	deviceNameLabel.AddCSSClass("heading")
	textBox.Append(deviceNameLabel)

	roomLabel := gtk.NewLabel(fmt.Sprintf("Score: %.0f", deviceData.Measurement.Score))
	roomLabel.SetHAlign(gtk.AlignStart)
	roomLabel.SetXAlign(0)
	roomLabel.AddCSSClass("dim-label")
	textBox.Append(roomLabel)

	mainBox.Append(textBox)
	row.SetChild(mainBox)

	mainBox.SetObjectProperty("cursor", "pointer")
	
	gesture := gtk.NewGestureClick()
	gesture.ConnectPressed(func(nPress int, x, y float64) {
		app.showDevicePage(index)
	})
	row.AddController(gesture)

	return row
}

func (app *App) createScoreCircle(score float64) *gtk.DrawingArea {
	area := gtk.NewDrawingArea()
	area.SetSizeRequest(60, 60)
	
	area.SetDrawFunc(func(area *gtk.DrawingArea, cr *cairo.Context, width, height int) {
		centerX := float64(width) / 2
		centerY := float64(height) / 2
		radius := math.Min(float64(width), float64(height))/2 - 4

		var r, g, b float64
		if score < 30 {
			r, g, b = 0.8, 0.2, 0.2
		} else if score < 75 {
			r, g, b = 0.9, 0.7, 0.1
		} else {
			r, g, b = 0.2, 0.7, 0.2
		}

		cr.SetSourceRGB(r, g, b)
		cr.Arc(centerX, centerY, radius, 0, 2*math.Pi)
		cr.Fill()

		cr.SetSourceRGB(1, 1, 1)
		cr.SelectFontFace("Sans", cairo.FontSlantNormal, cairo.FontWeightBold)
		cr.SetFontSize(16)
		
		text := fmt.Sprintf("%.0f", score)
		textExtents := cr.TextExtents(text)
		cr.MoveTo(centerX-textExtents.Width/2, centerY+textExtents.Height/2)
		cr.ShowText(text)
	})
	
	return area
}

func (app *App) showDevicePage(deviceIndex int) {
	deviceData := app.devices[deviceIndex]
	
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolled.SetVExpand(true)

	contentBox := gtk.NewBox(gtk.OrientationVertical, 24)
	contentBox.SetMarginTop(24)
	contentBox.SetMarginBottom(24)
	contentBox.SetMarginStart(24)
	contentBox.SetMarginEnd(24)

	deviceHeader := gtk.NewBox(gtk.OrientationHorizontal, 16)
	deviceHeader.SetHAlign(gtk.AlignCenter)

	scoreCircle := app.createScoreCircle(deviceData.Measurement.Score)
	deviceHeader.Append(scoreCircle)

	headerTextBox := gtk.NewBox(gtk.OrientationVertical, 4)
	headerTextBox.SetVAlign(gtk.AlignCenter)

	deviceTitle := gtk.NewLabel(deviceData.Device.Name)
	deviceTitle.AddCSSClass("title-1")
	headerTextBox.Append(deviceTitle)

	scoreLabel := gtk.NewLabel(fmt.Sprintf("Air Quality Score: %.0f", deviceData.Measurement.Score))
	scoreLabel.AddCSSClass("subtitle")
	headerTextBox.Append(scoreLabel)

	deviceHeader.Append(headerTextBox)
	contentBox.Append(deviceHeader)

	metricsGroup := adw.NewPreferencesGroup()
	metricsGroup.SetTitle("Current Measurements")

	metrics := []struct {
		name  string
		value float64
		unit  string
	}{
		{"Temperature", deviceData.Measurement.Temperature, "°C"},
		{"Humidity", deviceData.Measurement.Humidity, "%"},
		{"CO₂", deviceData.Measurement.CO2, "ppm"},
		{"VOC", deviceData.Measurement.VOC, "ppb"},
		{"PM2.5", deviceData.Measurement.PM25, "μg/m³"},
	}

	for _, metric := range metrics {
		row := adw.NewActionRow()
		row.SetTitle(metric.name)
		
		valueText := app.formatValue(metric.value, metric.unit)
		valueLabel := gtk.NewLabel(valueText)
		valueLabel.AddCSSClass("numeric")
		row.AddSuffix(valueLabel)
		
		metricsGroup.Add(row)
	}

	contentBox.Append(metricsGroup)

	deviceInfoGroup := adw.NewPreferencesGroup()
	deviceInfoGroup.SetTitle("Device Information")

	deviceInfoItems := []struct {
		title string
		value string
	}{
		{"Device Type", deviceData.Device.DeviceType},
		{"Serial Number", deviceData.Device.SerialNumber},
		{"IP Address", deviceData.Device.IPAddress},
		{"Last Seen", deviceData.Device.LastSeen.Format("Jan 2, 15:04")},
		{"Last Measurement", deviceData.Measurement.Timestamp.Format("Jan 2, 15:04")},
	}

	for _, item := range deviceInfoItems {
		row := adw.NewActionRow()
		row.SetTitle(item.title)
		
		valueLabel := gtk.NewLabel(item.value)
		valueLabel.AddCSSClass("dim-label")
		row.AddSuffix(valueLabel)
		
		deviceInfoGroup.Add(row)
	}

	contentBox.Append(deviceInfoGroup)
	scrolled.SetChild(contentBox)

	pageName := fmt.Sprintf("device-%d", deviceIndex)
	app.stack.AddNamed(scrolled, pageName)
	app.stack.SetVisibleChildName(pageName)
	
	app.mainWindow.SetTitle(deviceData.Device.Name + " - Air Quality")
	app.backButton.SetVisible(true)
	app.settingsButton.SetVisible(false)
}

func (app *App) showIndexPage() {
	app.stack.SetVisibleChildName("index")
	app.mainWindow.SetTitle("Air Monitor")
	app.backButton.SetVisible(false)
	app.settingsButton.SetVisible(true)
}

func (app *App) showSettingsPage() {
	app.stack.SetVisibleChildName("settings")
	app.mainWindow.SetTitle("Settings")
	app.backButton.SetVisible(true)
	app.settingsButton.SetVisible(false)
}

func (app *App) setupSettingsPage() {
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolled.SetVExpand(true)

	contentBox := gtk.NewBox(gtk.OrientationVertical, 24)
	contentBox.SetMarginTop(24)
	contentBox.SetMarginBottom(24)
	contentBox.SetMarginStart(24)
	contentBox.SetMarginEnd(24)

	titleLabel := gtk.NewLabel("Settings")
	titleLabel.AddCSSClass("title-1")
	titleLabel.SetHAlign(gtk.AlignStart)
	contentBox.Append(titleLabel)

	placeholderGroup := adw.NewPreferencesGroup()
	placeholderGroup.SetTitle("General")
	placeholderGroup.SetDescription("Application settings will be available here")

	placeholderRow := adw.NewActionRow()
	placeholderRow.SetTitle("Settings")
	placeholderRow.SetSubtitle("More settings will be added here in the future")
	placeholderGroup.Add(placeholderRow)

	contentBox.Append(placeholderGroup)
	scrolled.SetChild(contentBox)
	app.stack.AddNamed(scrolled, "settings")
}

func (app *App) formatValue(value float64, unit string) string {
	if value == float64(int(value)) {
		return fmt.Sprintf("%d %s", int(value), unit)
	}
	return fmt.Sprintf("%.1f %s", value, unit)
}

func (app *App) Run() int {
	return app.Application.Run(nil)
}
