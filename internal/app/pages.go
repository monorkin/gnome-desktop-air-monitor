package app

import (
	"fmt"

	adw "github.com/diamondburned/gotk4-adwaita/pkg/adw"
	gtk "github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func (app *App) setupIndexPage() {
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolled.SetVExpand(true)

	app.indexListBox = gtk.NewListBox()
	app.indexListBox.SetSelectionMode(gtk.SelectionNone)
	app.indexListBox.AddCSSClass("boxed-list")
	app.indexListBox.SetMarginTop(24)
	app.indexListBox.SetMarginBottom(24)
	app.indexListBox.SetMarginStart(24)
	app.indexListBox.SetMarginEnd(24)

	app.populateIndexPage()

	scrolled.SetChild(app.indexListBox)
	app.stack.AddNamed(scrolled, "index")
	app.stack.SetVisibleChildName("index")
}

func (app *App) populateIndexPage() {
	// Clear existing rows
	for app.indexListBox.FirstChild() != nil {
		app.indexListBox.Remove(app.indexListBox.FirstChild())
	}

	if len(app.devices) == 0 {
		// Show empty state
		emptyBox := gtk.NewBox(gtk.OrientationVertical, 12)
		emptyBox.SetHAlign(gtk.AlignCenter)
		emptyBox.SetVAlign(gtk.AlignCenter)
		emptyBox.SetMarginTop(48)
		emptyBox.SetMarginBottom(48)

		emptyIcon := gtk.NewLabel("📡")
		emptyIcon.AddCSSClass("title-1")
		emptyBox.Append(emptyIcon)

		emptyLabel := gtk.NewLabel("No devices found")
		emptyLabel.AddCSSClass("title-2")
		emptyBox.Append(emptyLabel)

		emptyDescription := gtk.NewLabel("Devices will appear here when discovered on your network")
		emptyDescription.AddCSSClass("dim-label")
		emptyDescription.SetWrap(true)
		emptyDescription.SetJustify(gtk.JustifyCenter)
		emptyBox.Append(emptyDescription)

		app.indexListBox.Append(emptyBox)
		return
	}

	for i, deviceData := range app.devices {
		row := app.createDeviceRow(deviceData, i)
		app.indexListBox.Append(row)
	}
}

func (app *App) refreshIndexPage() {
	if app.indexListBox != nil {
		app.populateIndexPage()
	}
}

func (app *App) showDevicePage(deviceIndex int) {
	deviceData := app.devices[deviceIndex]
	
	// Track the currently shown device
	app.currentDeviceSerial = deviceData.Device.SerialNumber

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
	
	// Remove existing page if it exists to avoid duplicate names
	existingPage := app.stack.ChildByName(pageName)
	if existingPage != nil {
		app.stack.Remove(existingPage)
	}
	
	app.stack.AddNamed(scrolled, pageName)
	app.stack.SetVisibleChildName(pageName)

	app.mainWindow.SetTitle(deviceData.Device.Name + " - Air Quality")
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

func (app *App) showIndexPage() {
	app.stack.SetVisibleChildName("index")
	app.mainWindow.SetTitle("Air Monitor")
	app.backButton.SetVisible(false)
	app.settingsButton.SetVisible(true)
	// Clear current device tracking
	app.currentDeviceSerial = ""
}

func (app *App) showSettingsPage() {
	app.stack.SetVisibleChildName("settings")
	app.mainWindow.SetTitle("Settings")
	app.backButton.SetVisible(true)
	app.settingsButton.SetVisible(false)
	// Clear current device tracking
	app.currentDeviceSerial = ""
}

// refreshCurrentDevicePage refreshes the currently shown device page if one is displayed
func (app *App) refreshCurrentDevicePage() {
	if app.currentDeviceSerial == "" {
		return // No device page is currently shown
	}

	// Find the device by serial number
	for i, deviceData := range app.devices {
		if deviceData.Device.SerialNumber == app.currentDeviceSerial {
			// Re-show the device page with updated data
			app.showDevicePage(i)
			return
		}
	}

	// Device not found (might have been removed), go back to index
	app.showIndexPage()
}
