package app

import (
	"fmt"

	adw "github.com/diamondburned/gotk4-adwaita/pkg/adw"
	gdk "github.com/diamondburned/gotk4/pkg/gdk/v4"
	glib "github.com/diamondburned/gotk4/pkg/glib/v2"
	gtk "github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/database"
	"github.com/monorkin/gnome-desktop-air-monitor/internal/models"
)

func (app *App) showDevicePage(deviceIndex int) {
	// Fetch devices from database
	devices, err := app.getDevicesWithMeasurements()
	if err != nil || deviceIndex >= len(devices) {
		app.showIndexPage()
		return
	}

	deviceData := devices[deviceIndex]

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

	// Create clickable device name with inline editing
	app.setupEditableDeviceName(headerTextBox, &deviceData, deviceIndex)

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

// refreshCurrentDevicePage refreshes the currently shown device page if one is displayed
func (app *App) refreshCurrentDevicePage() {
	if app.currentDeviceSerial == "" {
		return // No device page is currently shown
	}

	// Fetch devices from database
	devices, err := app.getDevicesWithMeasurements()
	if err != nil {
		app.showIndexPage()
		return
	}

	// Find the device by serial number
	for i, deviceData := range devices {
		if deviceData.Device.SerialNumber == app.currentDeviceSerial {
			// Re-show the device page with updated data
			app.showDevicePage(i)
			return
		}
	}

	// Device not found (might have been removed), go back to index
	app.showIndexPage()
}

// setupEditableDeviceName creates an editable device name widget
func (app *App) setupEditableDeviceName(container *gtk.Box, deviceData *DeviceWithMeasurement, deviceIndex int) {
	// Create a stack to switch between label and entry
	nameStack := gtk.NewStack()
	nameStack.SetTransitionType(gtk.StackTransitionTypeSlideUpDown)
	nameStack.SetTransitionDuration(200)
	
	// Create entry widget first so it's accessible in the click handler
	nameEntry := gtk.NewEntry()
	nameEntry.SetText(deviceData.Device.Name)
	nameEntry.SetHExpand(true)
	nameEntry.AddCSSClass("title-1")
	
	// Label view (default)
	labelBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	labelBox.SetHAlign(gtk.AlignStart)
	
	deviceNameLabel := gtk.NewLabel(deviceData.Device.Name)
	deviceNameLabel.AddCSSClass("title-1")
	labelBox.Append(deviceNameLabel)
	
	// Add a small edit icon to indicate it's clickable
	editIcon := gtk.NewLabel("✏️")
	editIcon.SetOpacity(0.7)
	labelBox.Append(editIcon)
	
	// Make label clickable
	labelGesture := gtk.NewGestureClick()
	labelGesture.ConnectPressed(func(nPress int, x, y float64) {
		if nPress == 1 { // Single click
			app.isEditingDeviceName = true
			nameStack.SetVisibleChildName("edit")
			nameEntry.GrabFocus() // Focus the entry field
		}
	})
	labelBox.AddController(labelGesture)
	
	nameStack.AddNamed(labelBox, "view")
	
	// Entry view (for editing)
	editBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	editBox.Append(nameEntry)
	
	// Save button
	saveButton := gtk.NewButtonFromIconName("object-select-symbolic")
	saveButton.AddCSSClass("suggested-action")
	saveButton.SetTooltipText("Save")
	saveButton.ConnectClicked(func() {
		newName := nameEntry.Text()
		if newName != "" && newName != deviceData.Device.Name {
			app.updateDeviceName(deviceData.Device.ID, newName, deviceIndex)
		} else {
			// Cancel edit - switch back to view
			app.isEditingDeviceName = false
			nameStack.SetVisibleChildName("view")
		}
	})
	editBox.Append(saveButton)
	
	// Cancel button
	cancelButton := gtk.NewButtonFromIconName("process-stop-symbolic")
	cancelButton.SetTooltipText("Cancel")
	cancelButton.ConnectClicked(func() {
		nameEntry.SetText(deviceData.Device.Name) // Reset text
		app.isEditingDeviceName = false
		nameStack.SetVisibleChildName("view")
	})
	editBox.Append(cancelButton)
	
	// Handle Enter key to save
	nameEntry.ConnectActivate(func() {
		saveButton.Activate()
	})
	
	// Handle Escape key to cancel
	keyController := gtk.NewEventControllerKey()
	keyController.ConnectKeyPressed(func(keyval uint, keycode uint, state gdk.ModifierType) bool {
		if keyval == gdk.KEY_Escape {
			cancelButton.Activate()
			return true
		}
		return false
	})
	nameEntry.AddController(keyController)
	
	nameStack.AddNamed(editBox, "edit")
	nameStack.SetVisibleChildName("view")
	
	container.Append(nameStack)
}

// updateDeviceName updates the device name in the database
func (app *App) updateDeviceName(deviceID uint, newName string, deviceIndex int) {
	app.logger.Info("Updating device name", "device_id", deviceID, "new_name", newName)
	
	// Update device name in database
	err := database.DB.Model(&models.Device{}).Where("id = ?", deviceID).Update("name", newName).Error
	if err != nil {
		app.logger.Error("Failed to update device name", "device_id", deviceID, "error", err)
		app.isEditingDeviceName = false // Clear flag so UI can refresh normally
		return
	}
	
	app.logger.Info("Device name updated successfully", "device_id", deviceID, "new_name", newName)
	
	// Clear editing flag and refresh the UI
	app.isEditingDeviceName = false
	app.refreshDevicesFromDatabaseSafe()
	
	// The page will be refreshed automatically, but we need to update the window title
	glib.IdleAdd(func() bool {
		app.mainWindow.SetTitle(newName + " - Air Quality")
		return false
	})
}
