package app

import (
	gtk "github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// IndexPageState holds all state related to the device index page
type IndexPageState struct {
	listBox *gtk.ListBox
}

// App wrapper methods for backward compatibility
func (app *App) setupIndexPage() {
	app.indexPage.setupIndexPage(app)
}

func (app *App) populateIndexPage() {
	app.indexPage.populateIndexPage(app)
}

func (app *App) refreshIndexPage() {
	app.indexPage.refreshIndexPage(app)
}

func (app *App) showIndexPage() {
	app.indexPage.showIndexPage(app)
}

// IndexPageState methods
func (ip *IndexPageState) setupIndexPage(app *App) {
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolled.SetVExpand(true)

	ip.listBox = gtk.NewListBox()
	ip.listBox.SetSelectionMode(gtk.SelectionNone)
	ip.listBox.AddCSSClass("boxed-list")
	ip.listBox.SetMarginTop(24)
	ip.listBox.SetMarginBottom(24)
	ip.listBox.SetMarginStart(24)
	ip.listBox.SetMarginEnd(24)

	ip.populateIndexPage(app)

	scrolled.SetChild(ip.listBox)
	app.stack.AddNamed(scrolled, "index")
	app.stack.SetVisibleChildName("index")
}

func (ip *IndexPageState) populateIndexPage(app *App) {
	// Clear existing rows
	for ip.listBox.FirstChild() != nil {
		ip.listBox.Remove(ip.listBox.FirstChild())
	}

	// Fetch devices from database
	devices, err := app.getDevicesWithMeasurements()
	if err != nil {
		// Show error state
		errorBox := gtk.NewBox(gtk.OrientationVertical, 12)
		errorBox.SetHAlign(gtk.AlignCenter)
		errorBox.SetVAlign(gtk.AlignCenter)
		errorBox.SetMarginTop(48)
		errorBox.SetMarginBottom(48)

		errorIcon := gtk.NewLabel("⚠️")
		errorIcon.AddCSSClass("title-1")
		errorBox.Append(errorIcon)

		errorLabel := gtk.NewLabel("Error loading devices")
		errorLabel.AddCSSClass("title-2")
		errorBox.Append(errorLabel)

		errorDescription := gtk.NewLabel("Failed to load devices from database")
		errorDescription.AddCSSClass("dim-label")
		errorDescription.SetWrap(true)
		errorDescription.SetJustify(gtk.JustifyCenter)
		errorBox.Append(errorDescription)

		ip.listBox.Append(errorBox)
		return
	}

	if len(devices) == 0 {
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

		ip.listBox.Append(emptyBox)
		return
	}

	for i, deviceData := range devices {
		row := app.createDeviceRow(deviceData, i)
		ip.listBox.Append(row)
	}
}

func (ip *IndexPageState) refreshIndexPage(app *App) {
	if ip.listBox != nil {
		ip.populateIndexPage(app)
	}
}

func (ip *IndexPageState) showIndexPage(app *App) {
	app.stack.SetVisibleChildName("index")
	app.mainWindow.SetTitle("Air Monitor")
	app.backButton.SetVisible(false)
	app.settingsButton.SetVisible(true)
	// Clear device page state when leaving device page
	app.devicePage.clearState()
}