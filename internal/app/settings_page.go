package app

import (
	"fmt"

	adw "github.com/diamondburned/gotk4-adwaita/pkg/adw"
	glib "github.com/diamondburned/gotk4/pkg/glib/v2"
	gtk "github.com/diamondburned/gotk4/pkg/gtk/v4"
	pango "github.com/diamondburned/gotk4/pkg/pango"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/config"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/database"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/globals"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/licenses"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/version"
)

// SettingsPageState holds all state related to the settings page
type SettingsPageState struct {
	// UI widget references for potential future use
	visibilitySwitch    *gtk.Switch
	deviceDropdown      *gtk.DropDown
	retentionSpinButton *gtk.SpinButton
}

// SettingsPageState methods
func (sp *SettingsPageState) setup(app *App) {
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolled.SetVExpand(true)

	contentBox := gtk.NewBox(gtk.OrientationVertical, 24)
	contentBox.SetMarginTop(24)
	contentBox.SetMarginBottom(24)
	contentBox.SetMarginStart(36)
	contentBox.SetMarginEnd(36)

	titleLabel := gtk.NewLabel("Settings")
	titleLabel.AddCSSClass("title-1")
	titleLabel.SetHAlign(gtk.AlignStart)
	contentBox.Append(titleLabel)

	// Shell Extension settings group
	shellGroup := adw.NewPreferencesGroup()
	shellGroup.SetTitle("Status bar")
	shellGroup.SetMarginStart(12)
	shellGroup.SetMarginEnd(12)
	shellGroup.SetDescription("Configure the status bar indicator")

	// Shell extension visibility toggle
	visibilityRow := adw.NewActionRow()
	visibilityRow.SetTitle("Show Status Bar Indicator")
	visibilityRow.SetSubtitle("Display air quality information in the top bar")
	visibilityRow.AddCSSClass("padded-row")

	sp.visibilitySwitch = gtk.NewSwitch()
	sp.visibilitySwitch.SetVAlign(gtk.AlignCenter)
	visibilityRow.AddSuffix(sp.visibilitySwitch)
	visibilityRow.SetActivatableWidget(sp.visibilitySwitch)

	// Set initial state and connect to changes
	sp.setupToggle(app)

	shellGroup.Add(visibilityRow)

	// Device selection row
	deviceRow := adw.NewActionRow()
	deviceRow.SetTitle("Device")
	deviceRow.SetSubtitle("Choose which device to display in the shell extension")
	deviceRow.AddCSSClass("padded-row")

	// Create dropdown for device selection
	sp.setupDropdown(app, deviceRow)

	shellGroup.Add(deviceRow)
	contentBox.Append(shellGroup)

	// Data Retention settings group
	dataGroup := adw.NewPreferencesGroup()
	dataGroup.SetTitle("Data Management")
	dataGroup.SetDescription("Configure data storage and retention policies")
	dataGroup.SetMarginStart(12)
	dataGroup.SetMarginEnd(12)

	// Data retention period row
	retentionRow := adw.NewActionRow()
	retentionRow.SetTitle("Data Retention Period")
	retentionRow.SetSubtitle("Number of days to keep measurement data")
	retentionRow.AddCSSClass("padded-row")

	// Create spin button for retention period
	retentionAdjustment := gtk.NewAdjustment(float64(globals.Settings.DataRetentionPeriod), 1, 365, 1, 7, 0)
	sp.retentionSpinButton = gtk.NewSpinButton(retentionAdjustment, 1, 0)
	sp.retentionSpinButton.SetVAlign(gtk.AlignCenter)
	sp.retentionSpinButton.SetValue(float64(globals.Settings.DataRetentionPeriod))

	// Connect to value changes
	sp.retentionSpinButton.ConnectValueChanged(func() {
		sp.onRetentionChanged(app, int(sp.retentionSpinButton.Value()))
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
	sizeRow.AddCSSClass("padded-row")

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

	// About/License settings group
	aboutGroup := adw.NewPreferencesGroup()
	aboutGroup.SetTitle("About")
	aboutGroup.SetDescription("Application information and legal notices")
	aboutGroup.SetMarginStart(12)
	aboutGroup.SetMarginEnd(12)

	// Version row
	versionRow := adw.NewActionRow()
	versionRow.SetTitle("Version")
	versionRow.SetSubtitle("Current application version")
	versionRow.AddCSSClass("padded-row")

	versionLabel := gtk.NewLabel(version.GetVersion())
	versionLabel.AddCSSClass("dim-label")
	versionLabel.SetVAlign(gtk.AlignCenter)
	versionLabel.SetSelectable(true)
	versionRow.AddSuffix(versionLabel)
	aboutGroup.Add(versionRow)

	// Config path row
	configPathRow := adw.NewActionRow()
	configPathRow.SetTitle("Configuration File")
	configPathRow.SetSubtitle("Location of the settings file")
	configPathRow.AddCSSClass("padded-row")

	configPathLabel := gtk.NewLabel(config.DefaultSettingsPath())
	configPathLabel.AddCSSClass("dim-label")
	configPathLabel.SetVAlign(gtk.AlignCenter)
	configPathLabel.SetSelectable(true)
	configPathLabel.SetEllipsize(pango.EllipsizeMiddle)
	configPathRow.AddSuffix(configPathLabel)
	aboutGroup.Add(configPathRow)

	// Database path row
	dbPathRow := adw.NewActionRow()
	dbPathRow.SetTitle("Database File")
	dbPathRow.SetSubtitle("Location of the measurement data")
	dbPathRow.AddCSSClass("padded-row")

	dbPathLabel := gtk.NewLabel(config.DBPath())
	dbPathLabel.AddCSSClass("dim-label")
	dbPathLabel.SetVAlign(gtk.AlignCenter)
	dbPathLabel.SetSelectable(true)
	dbPathLabel.SetEllipsize(pango.EllipsizeMiddle)
	dbPathRow.AddSuffix(dbPathLabel)
	aboutGroup.Add(dbPathRow)

	// Project license row
	projectLicenseRow := adw.NewActionRow()
	projectLicenseRow.SetTitle("Project License")
	projectLicenseRow.SetSubtitle("View the license for this application")
	projectLicenseRow.AddCSSClass("padded-row")

	projectLicenseButton := gtk.NewButton()
	projectLicenseButton.SetLabel("View")
	projectLicenseButton.SetVAlign(gtk.AlignCenter)
	projectLicenseButton.ConnectClicked(func() {
		sp.showLicenseModal(app, "Project License", licenses.GetProjectLicense)
	})

	projectLicenseRow.AddSuffix(projectLicenseButton)
	aboutGroup.Add(projectLicenseRow)

	// Third party licenses row
	thirdPartyLicenseRow := adw.NewActionRow()
	thirdPartyLicenseRow.SetTitle("Third Party Licenses")
	thirdPartyLicenseRow.SetSubtitle("View licenses for bundled third-party libraries")
	thirdPartyLicenseRow.AddCSSClass("padded-row")

	thirdPartyLicenseButton := gtk.NewButton()
	thirdPartyLicenseButton.SetLabel("View")
	thirdPartyLicenseButton.SetVAlign(gtk.AlignCenter)
	thirdPartyLicenseButton.ConnectClicked(func() {
		sp.showLicenseModal(app, "Third Party Licenses", licenses.GetThirdPartyLicenses)
	})

	thirdPartyLicenseRow.AddSuffix(thirdPartyLicenseButton)
	aboutGroup.Add(thirdPartyLicenseRow)

	contentBox.Append(aboutGroup)

	scrolled.SetChild(contentBox)
	app.stack.AddNamed(scrolled, "settings")
}

func (sp *SettingsPageState) show(app *App) {
	app.stack.SetVisibleChildName("settings")
	app.mainWindow.SetTitle("Settings")
	app.backButton.SetVisible(true)
	app.settingsButton.SetVisible(false)
	// Clear device page state when leaving device page
	app.devicePage.clearState()
}

// setupDeviceDropdown creates and configures the device selection dropdown
func (sp *SettingsPageState) setupDropdown(app *App, deviceRow *adw.ActionRow) {
	// Create string list model for the dropdown
	stringList := gtk.NewStringList(nil)

	// Create dropdown
	sp.deviceDropdown = gtk.NewDropDown(stringList, nil)
	sp.deviceDropdown.SetHExpand(false)
	sp.deviceDropdown.SetVAlign(gtk.AlignCenter)

	// Add dropdown to the row
	deviceRow.AddSuffix(sp.deviceDropdown)

	// Load devices and populate dropdown
	sp.refreshDropdown(app, stringList)

	// Connect to selection changes
	sp.deviceDropdown.Connect("notify::selected", func() {
		selectedIndex := sp.deviceDropdown.Selected()
		sp.onSelectionChanged(app, uint32(selectedIndex), stringList)
	})
}

// refreshDeviceDropdown refreshes the device dropdown with current devices
func (sp *SettingsPageState) refreshDropdown(app *App, stringList *gtk.StringList) {
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
		if globals.Settings.StatusBarDeviceSerialNumber != nil &&
			deviceData.Device.SerialNumber == *globals.Settings.StatusBarDeviceSerialNumber {
			selectedIndex = uint32(i + 1) // +1 because of "No device selected" option
		}
	}

	// Set the current selection
	sp.deviceDropdown.SetSelected(uint(selectedIndex))
}

// onDeviceSelectionChanged handles device selection changes in the dropdown
func (sp *SettingsPageState) onSelectionChanged(app *App, selectedIndex uint32, stringList *gtk.StringList) {
	if selectedIndex == 0 {
		// "No device selected" option chosen
		globals.Settings.StatusBarDeviceSerialNumber = nil
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
			globals.Settings.StatusBarDeviceSerialNumber = &selectedSerial
			app.logger.Info("Device selected for status bar", "device_serial", selectedSerial)
		}
	}

	// Save settings
	err := globals.Settings.Save()
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
func (sp *SettingsPageState) setupToggle(app *App) {
	// Set initial state based on settings
	sp.visibilitySwitch.SetActive(globals.Settings.ShowShellExtension)

	// Connect to state changes
	sp.visibilitySwitch.Connect("state-set", func(state bool) bool {
		sp.onToggleChanged(app, state)
		return false // Allow the state change to proceed
	})
}

// onVisibilityToggleChanged handles changes to the shell extension visibility setting
func (sp *SettingsPageState) onToggleChanged(app *App, visible bool) {
	app.logger.Info("Shell extension visibility changed", "visible", visible)

	// Update settings
	globals.Settings.ShowShellExtension = visible

	// Save settings
	err := globals.Settings.Save()
	if err != nil {
		app.logger.Error("Failed to save visibility setting", "error", err)
		return
	}

	// Update shell extension
	if app.dbusService != nil {
		app.dbusService.EmitVisibilityChanged()
	}
}

// onRetentionPeriodChanged handles changes to the data retention period setting
func (sp *SettingsPageState) onRetentionChanged(app *App, days int) {
	app.logger.Info("Data retention period changed", "new_days", days, "old_days", globals.Settings.DataRetentionPeriod)

	// Update settings
	globals.Settings.DataRetentionPeriod = days

	// Save settings
	err := globals.Settings.Save()
	if err != nil {
		app.logger.Error("Failed to save retention period setting", "error", err)
		return
	}

	// Trigger immediate cleanup with new retention period
	app.cleanupOldMeasurements()
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

// showLicenseModal displays a modal dialog with license text
func (sp *SettingsPageState) showLicenseModal(app *App, title string, getLicense func() (string, error)) {
	content, err := getLicense()
	if err != nil {
		content = err.Error()
	}

	// Create dialog
	dialog := adw.NewMessageDialog(&app.mainWindow.Window, title, "")
	dialog.SetHeading(title)

	// Get current window size and set dialog to 3/4 of it
	width, height := app.mainWindow.DefaultSize()
	dialog.SetDefaultSize(int(float64(width)*0.75), int(float64(height)*0.75))

	// Create scrolled window for content
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolled.SetVExpand(true)
	scrolled.SetHExpand(true)

	// Create text view for license content
	textView := gtk.NewTextView()
	textView.SetEditable(false)
	textView.SetWrapMode(gtk.WrapWord)
	textView.SetMonospace(true)
	textView.SetMarginTop(12)
	textView.SetMarginBottom(12)
	textView.SetMarginStart(12)
	textView.SetMarginEnd(12)

	// Set text content
	buffer := textView.Buffer()
	buffer.SetText(content)

	scrolled.SetChild(textView)

	// Set the scrolled window as the extra child of the dialog
	dialog.SetExtraChild(scrolled)

	// Add close button
	dialog.AddResponse("close", "Close")
	dialog.SetDefaultResponse("close")

	// Connect response signal
	dialog.ConnectResponse(func(response string) {
		dialog.Destroy()
	})

	dialog.Present()
}
