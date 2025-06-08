package app

import (
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

		app.indexListBox.Append(errorBox)
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

		app.indexListBox.Append(emptyBox)
		return
	}

	for i, deviceData := range devices {
		row := app.createDeviceRow(deviceData, i)
		app.indexListBox.Append(row)
	}
}

func (app *App) refreshIndexPage() {
	if app.indexListBox != nil {
		app.populateIndexPage()
	}
}

func (app *App) showIndexPage() {
	app.stack.SetVisibleChildName("index")
	app.mainWindow.SetTitle("Air Monitor")
	app.backButton.SetVisible(false)
	app.settingsButton.SetVisible(true)
	// Clear current device tracking
	app.currentDeviceSerial = ""
}
