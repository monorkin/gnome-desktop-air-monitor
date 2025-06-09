package app

import (
	"fmt"

	adw "github.com/diamondburned/gotk4-adwaita/pkg/adw"
	glib "github.com/diamondburned/gotk4/pkg/glib/v2"
	gtk "github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/database"
)

// SettingsPageState holds all state related to the settings page
type SettingsPageState struct {
	// UI widget references for potential future use
	visibilitySwitch    *gtk.Switch
	deviceDropdown      *gtk.DropDown
	retentionSpinButton *gtk.SpinButton
}

// App wrapper methods for backward compatibility
func (app *App) setupSettingsPage() {
	app.settingsPage.setupSettingsPage(app)
}

func (app *App) showSettingsPage() {
	app.settingsPage.showSettingsPage(app)
}

// SettingsPageState methods
func (sp *SettingsPageState) setupSettingsPage(app *App) {
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

	sp.visibilitySwitch = gtk.NewSwitch()
	sp.visibilitySwitch.SetVAlign(gtk.AlignCenter)
	visibilityRow.AddSuffix(sp.visibilitySwitch)
	visibilityRow.SetActivatableWidget(sp.visibilitySwitch)

	// Set initial state and connect to changes
	sp.setupVisibilityToggle(app)

	shellGroup.Add(visibilityRow)

	// Device selection row
	deviceRow := adw.NewActionRow()
	deviceRow.SetTitle("Device")
	deviceRow.SetSubtitle("Choose which device to display in the shell extension")

	// Create dropdown for device selection
	sp.setupDeviceDropdown(app, deviceRow)

	shellGroup.Add(deviceRow)
	contentBox.Append(shellGroup)

	// Data Retention settings group
	dataGroup := adw.NewPreferencesGroup()
	dataGroup.SetTitle("Data Management")
	dataGroup.SetDescription("Configure data storage and retention policies")

	// Data retention period row
	retentionRow := adw.NewActionRow()
	retentionRow.SetTitle("Data Retention Period")
	retentionRow.SetSubtitle("Number of days to keep measurement data")

	// Create spin button for retention period
	retentionAdjustment := gtk.NewAdjustment(float64(settings.DataRetentionPeriod), 1, 365, 1, 7, 0)
	sp.retentionSpinButton = gtk.NewSpinButton(retentionAdjustment, 1, 0)
	sp.retentionSpinButton.SetVAlign(gtk.AlignCenter)
	sp.retentionSpinButton.SetValue(float64(settings.DataRetentionPeriod))

	// Connect to value changes
	sp.retentionSpinButton.ConnectValueChanged(func() {
		app.onRetentionPeriodChanged(int(sp.retentionSpinButton.Value()))
	})

	// Add suffix label for "days"
	suffixBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	suffixBox.Append(sp.retentionSpinButton)
	daysLabel := gtk.NewLabel("days")
	daysLabel.AddCSSClass("dim-label")
	daysLabel.SetVAlign(gtk.AlignCenter)
	suffixBox.Append(daysLabel)

	retentionRow.AddSuffix(suffixBox)
	dataGroup.Add(retentionRow)

	// Database size row
	sizeRow := adw.NewActionRow()
	sizeRow.SetTitle("Database Size")
	sizeRow.SetSubtitle("Current storage space used by measurement data")

	// Get and format database size
	sizeLabel := gtk.NewLabel("Calculating...")
	sizeLabel.AddCSSClass("dim-label")
	sizeLabel.SetVAlign(gtk.AlignCenter)

	// Load size asynchronously to avoid blocking UI
	go func() {
		if size, err := database.GetSize(); err == nil {
			sizeText := sp.formatFileSize(size)
			// Update UI from main thread
			glib.IdleAdd(func() bool {
				sizeLabel.SetText(sizeText)
				return false
			})
		} else {
			glib.IdleAdd(func() bool {
				sizeLabel.SetText("Error reading size")
				return false
			})
		}
	}()

	sizeRow.AddSuffix(sizeLabel)
	dataGroup.Add(sizeRow)

	contentBox.Append(dataGroup)

	scrolled.SetChild(contentBox)
	app.stack.AddNamed(scrolled, "settings")
}

func (sp *SettingsPageState) showSettingsPage(app *App) {
	app.stack.SetVisibleChildName("settings")
	app.mainWindow.SetTitle("Settings")
	app.backButton.SetVisible(true)
	app.settingsButton.SetVisible(false)
	// Clear device page state when leaving device page
	app.devicePage.clearState()
}

// setupDeviceDropdown creates and configures the device selection dropdown
func (sp *SettingsPageState) setupDeviceDropdown(app *App, deviceRow *adw.ActionRow) {
	// Create string list model for the dropdown
	stringList := gtk.NewStringList(nil)

	// Create dropdown
	sp.deviceDropdown = gtk.NewDropDown(stringList, nil)
	sp.deviceDropdown.SetHExpand(false)
	sp.deviceDropdown.SetVAlign(gtk.AlignCenter)

	// Add dropdown to the row
	deviceRow.AddSuffix(sp.deviceDropdown)

	// Load devices and populate dropdown
	sp.refreshDeviceDropdown(app, stringList)

	// Connect to selection changes
	sp.deviceDropdown.Connect("notify::selected", func() {
		selectedIndex := sp.deviceDropdown.Selected()
		sp.onDeviceSelectionChanged(app, uint32(selectedIndex), stringList)
	})
}

// refreshDeviceDropdown refreshes the device dropdown with current devices
func (sp *SettingsPageState) refreshDeviceDropdown(app *App, stringList *gtk.StringList) {
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
	sp.deviceDropdown.SetSelected(uint(selectedIndex))
}

// onDeviceSelectionChanged handles device selection changes in the dropdown
func (sp *SettingsPageState) onDeviceSelectionChanged(app *App, selectedIndex uint32, stringList *gtk.StringList) {
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
func (sp *SettingsPageState) setupVisibilityToggle(app *App) {
	// Set initial state based on settings
	sp.visibilitySwitch.SetActive(settings.ShowShellExtension)

	// Connect to state changes
	sp.visibilitySwitch.Connect("state-set", func(state bool) bool {
		sp.onVisibilityToggleChanged(app, state)
		return false // Allow the state change to proceed
	})
}

// onVisibilityToggleChanged handles changes to the shell extension visibility setting
func (sp *SettingsPageState) onVisibilityToggleChanged(app *App, visible bool) {
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

// formatFileSize formats bytes into a human-readable string
func (sp *SettingsPageState) formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}