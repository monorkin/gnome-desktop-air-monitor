package app

import (
	adw "github.com/diamondburned/gotk4-adwaita/pkg/adw"
	gtk "github.com/diamondburned/gotk4/pkg/gtk/v4"
)

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

	// Shell Extension settings group
	shellGroup := adw.NewPreferencesGroup()
	shellGroup.SetTitle("Status bar")
	// shellGroup.SetDescription("Configure the status bar indicator")

	// Shell extension visibility toggle
	visibilityRow := adw.NewActionRow()
	visibilityRow.SetTitle("Show Status Bar Indicator")
	visibilityRow.SetSubtitle("Display air quality information in the top bar")

	visibilitySwitch := gtk.NewSwitch()
	visibilitySwitch.SetVAlign(gtk.AlignCenter)
	visibilityRow.AddSuffix(visibilitySwitch)
	visibilityRow.SetActivatableWidget(visibilitySwitch)

	// Set initial state and connect to changes
	app.setupVisibilityToggle(visibilitySwitch)

	shellGroup.Add(visibilityRow)

	// Device selection row
	deviceRow := adw.NewActionRow()
	deviceRow.SetTitle("Device")
	deviceRow.SetSubtitle("Choose which device to display in the shell extension")

	// Create dropdown for device selection
	app.setupDeviceDropdown(deviceRow)

	shellGroup.Add(deviceRow)
	contentBox.Append(shellGroup)
	scrolled.SetChild(contentBox)
	app.stack.AddNamed(scrolled, "settings")
}

func (app *App) showSettingsPage() {
	app.stack.SetVisibleChildName("settings")
	app.mainWindow.SetTitle("Settings")
	app.backButton.SetVisible(true)
	app.settingsButton.SetVisible(false)
	// Clear current device tracking
	app.currentDeviceSerial = ""
}
