package app

import (
	"fmt"

	adw "github.com/diamondburned/gotk4-adwaita/pkg/adw"
	glib "github.com/diamondburned/gotk4/pkg/glib/v2"
	gtk "github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/database"
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
	retentionSpinButton := gtk.NewSpinButton(retentionAdjustment, 1, 0)
	retentionSpinButton.SetVAlign(gtk.AlignCenter)
	retentionSpinButton.SetValue(float64(settings.DataRetentionPeriod))

	// Connect to value changes
	retentionSpinButton.ConnectValueChanged(func() {
		app.onRetentionPeriodChanged(int(retentionSpinButton.Value()))
	})

	// Add suffix label for "days"
	suffixBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	suffixBox.Append(retentionSpinButton)
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
			sizeText := formatFileSize(size)
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

func (app *App) showSettingsPage() {
	app.stack.SetVisibleChildName("settings")
	app.mainWindow.SetTitle("Settings")
	app.backButton.SetVisible(true)
	app.settingsButton.SetVisible(false)
	// Clear current device tracking
	app.currentDeviceSerial = ""
	app.currentGraphState = nil // Clear graph state when leaving device page
	app.currentScrollPosition = 0 // Reset scroll position
	app.currentDeviceScrolled = nil // Clear reused scrolled window
}

// formatFileSize formats bytes into a human-readable string
func formatFileSize(bytes int64) string {
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
