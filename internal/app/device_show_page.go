package app

import (
	"fmt"
	"time"

	adw "github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/cairo"
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
	
	// Get vertical adjustment for scroll position management
	vAdjustment := scrolled.VAdjustment()
	
	// Save scroll position when it changes
	vAdjustment.ConnectValueChanged(func() {
		if app.currentDeviceSerial == deviceData.Device.SerialNumber {
			app.currentScrollPosition = vAdjustment.Value()
		}
	})

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

	// Add 24-hour graph with navigation
	app.addMeasurementGraph(contentBox, &deviceData)

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

	// Restore scroll position after a brief delay to ensure UI is rendered
	if app.currentScrollPosition > 0 {
		glib.IdleAdd(func() bool {
			vAdjustment.SetValue(app.currentScrollPosition)
			return false
		})
	}

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

// MetricInfo holds display information for each metric
type MetricInfo struct {
	Name  string
	Unit  string
	Color [3]float64 // RGB values 0-1
}

// getMetricInfo returns display information for each metric type
func getMetricInfo() map[MetricType]MetricInfo {
	return map[MetricType]MetricInfo{
		MetricTemperature: {"Temperature", "°C", [3]float64{0.96, 0.47, 0.24}}, // Orange
		MetricHumidity:    {"Humidity", "%", [3]float64{0.20, 0.74, 0.96}},     // Blue
		MetricCO2:         {"CO₂", "ppm", [3]float64{0.95, 0.61, 0.23}},        // Yellow-Orange
		MetricVOC:         {"VOC", "ppb", [3]float64{0.58, 0.75, 0.33}},        // Green
		MetricPM25:        {"PM2.5", "μg/m³", [3]float64{0.88, 0.32, 0.43}},    // Red
		MetricScore:       {"Score", "", [3]float64{0.45, 0.67, 0.89}},         // Light Blue
	}
}

// addMeasurementGraph creates and adds the measurement graph widget
func (app *App) addMeasurementGraph(container *gtk.Box, deviceData *DeviceWithMeasurement) {
	graphGroup := adw.NewPreferencesGroup()
	graphGroup.SetTitle("24-Hour Trends")

	// Reuse existing graph state if available, otherwise create new
	var graphState *GraphState
	if app.currentGraphState != nil && app.currentGraphState.device.Device.SerialNumber == deviceData.Device.SerialNumber {
		// Reuse existing state but update device data
		graphState = app.currentGraphState
		graphState.device = deviceData
		// Clear button references since we're recreating the UI
		graphState.metricButtons = make(map[MetricType]*gtk.Button)
	} else {
		// Create new graph state
		graphState = &GraphState{
			selectedMetric: MetricScore, // Default to air quality score
			timeOffset:     0,           // Start with current time
			device:         deviceData,
			metricButtons:  make(map[MetricType]*gtk.Button),
		}
		app.currentGraphState = graphState
	}

	// Metric selector buttons
	buttonRow := gtk.NewBox(gtk.OrientationHorizontal, 8)
	buttonRow.SetHAlign(gtk.AlignCenter)
	buttonRow.SetMarginTop(12)
	buttonRow.SetMarginBottom(12)

	// Create buttons in a consistent order
	metricOrder := []MetricType{MetricScore, MetricTemperature, MetricHumidity, MetricCO2, MetricVOC, MetricPM25}
	metricInfos := getMetricInfo()
	
	for _, metricType := range metricOrder {
		info := metricInfos[metricType]
		button := gtk.NewButton()
		button.SetLabel(info.Name)
		button.AddCSSClass("pill")

		if metricType == graphState.selectedMetric {
			button.AddCSSClass("suggested-action")
		}

		// Store button reference for styling updates
		graphState.metricButtons[metricType] = button

		// Capture the metric type for the closure
		currentMetric := metricType
		button.ConnectClicked(func() {
			app.selectMetric(graphState, currentMetric)
		})

		buttonRow.Append(button)
	}

	// Time navigation controls
	navRow := gtk.NewBox(gtk.OrientationHorizontal, 16)
	navRow.SetHAlign(gtk.AlignCenter)
	navRow.SetMarginBottom(12)

	// Left arrow (go back 8 hours)
	leftButton := gtk.NewButtonFromIconName("go-previous-symbolic")
	leftButton.SetTooltipText("Go back 8 hours")
	leftButton.ConnectClicked(func() {
		app.navigateTime(graphState, -8*time.Hour)
	})
	navRow.Append(leftButton)

	// Time label
	graphState.timeLabel = gtk.NewLabel(app.getTimeWindowLabel(graphState.timeOffset))
	graphState.timeLabel.AddCSSClass("caption")
	navRow.Append(graphState.timeLabel)

	// Right arrow (go forward 8 hours, but not beyond current time)
	rightButton := gtk.NewButtonFromIconName("go-next-symbolic")
	rightButton.SetTooltipText("Go forward 8 hours")
	rightButton.ConnectClicked(func() {
		app.navigateTime(graphState, 8*time.Hour)
	})
	navRow.Append(rightButton)

	// Graph drawing area
	graphState.drawingArea = gtk.NewDrawingArea()
	graphState.drawingArea.SetSizeRequest(600, 300)
	graphState.drawingArea.SetHExpand(true)
	graphState.drawingArea.SetVExpand(false)

	graphState.drawingArea.SetDrawFunc(func(area *gtk.DrawingArea, cr *cairo.Context, width, height int) {
		app.drawGraph(cr, graphState, width, height)
	})

	// Assemble the graph widget
	graphBox := gtk.NewBox(gtk.OrientationVertical, 8)
	graphBox.Append(buttonRow)
	graphBox.Append(navRow)
	graphBox.Append(graphState.drawingArea)

	graphGroup.Add(graphBox)
	container.Append(graphGroup)

	// Store reference for updates (you might want to add this to App struct if needed)
	// Initial draw will happen automatically
}

// selectMetric changes the selected metric and updates the graph
func (app *App) selectMetric(graphState *GraphState, metricType MetricType) {
	// Update button styles - remove suggested-action from all buttons
	for _, button := range graphState.metricButtons {
		button.RemoveCSSClass("suggested-action")
	}
	
	// Add suggested-action to the selected button
	if selectedButton, exists := graphState.metricButtons[metricType]; exists {
		selectedButton.AddCSSClass("suggested-action")
	}
	
	// Update the selected metric
	graphState.selectedMetric = metricType

	// Redraw graph
	graphState.drawingArea.QueueDraw()
}

// navigateTime moves the time window and updates the graph
func (app *App) navigateTime(graphState *GraphState, deltaTime time.Duration) {
	newOffset := graphState.timeOffset + deltaTime

	// Don't allow going into the future
	if newOffset > 0 {
		newOffset = 0
	}

	// Don't allow going too far back (e.g., more than 7 days)
	maxBack := -7 * 24 * time.Hour
	if newOffset < maxBack {
		newOffset = maxBack
	}

	graphState.timeOffset = newOffset

	// Update time label
	if graphState.timeLabel != nil {
		graphState.timeLabel.SetText(app.getTimeWindowLabel(newOffset))
	}
	
	// Redraw the graph
	graphState.drawingArea.QueueDraw()
}

// getTimeWindowLabel returns a human-readable label for the current time window
func (app *App) getTimeWindowLabel(offset time.Duration) string {
	if offset == 0 {
		return "Last 24 hours"
	}

	hoursAgo := int(-offset.Hours())
	if hoursAgo < 24 {
		return fmt.Sprintf("%d hours ago - now", hoursAgo)
	}

	daysAgo := hoursAgo / 24
	remainingHours := hoursAgo % 24

	if remainingHours == 0 {
		if daysAgo == 1 {
			return "1 day ago - now"
		}
		return fmt.Sprintf("%d days ago - now", daysAgo)
	}

	return fmt.Sprintf("%dd %dh ago - now", daysAgo, remainingHours)
}

// drawGraph renders the measurement graph
func (app *App) drawGraph(cr *cairo.Context, graphState *GraphState, width, height int) {
	// Set background
	cr.SetSourceRGB(1, 1, 1) // White background
	cr.Paint()

	// Graph margins
	marginLeft, marginRight := 60, 20
	marginTop, marginBottom := 20, 40

	graphWidth := width - marginLeft - marginRight
	graphHeight := height - marginTop - marginBottom

	if graphWidth <= 0 || graphHeight <= 0 {
		return
	}

	// Get measurements for the current time window
	measurements := app.getMeasurementsForTimeWindow(graphState.device.Device.ID, graphState.timeOffset)
	if len(measurements) == 0 {
		app.drawNoDataMessage(cr, width, height)
		return
	}

	// Get metric info
	metricInfos := getMetricInfo()
	metricInfo := metricInfos[graphState.selectedMetric]

	// Extract values for the selected metric
	values := make([]float64, len(measurements))
	times := make([]time.Time, len(measurements))

	for i, m := range measurements {
		times[i] = m.Timestamp
		switch graphState.selectedMetric {
		case MetricTemperature:
			values[i] = m.Temperature
		case MetricHumidity:
			values[i] = m.Humidity
		case MetricCO2:
			values[i] = m.CO2
		case MetricVOC:
			values[i] = m.VOC
		case MetricPM25:
			values[i] = m.PM25
		case MetricScore:
			values[i] = m.Score
		}
	}

	// Find value range
	minVal, maxVal := values[0], values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	// Add some padding to the range
	range_ := maxVal - minVal
	if range_ < 0.1 { // Avoid division by zero for constant values
		range_ = 1.0
	}
	padding := range_ * 0.1
	minVal -= padding
	maxVal += padding

	// Time range
	endTime := time.Now().Add(graphState.timeOffset)
	startTime := endTime.Add(-24 * time.Hour)

	// Draw grid and axes
	app.drawGridAndAxes(cr, marginLeft, marginTop, graphWidth, graphHeight,
		startTime, endTime, minVal, maxVal, metricInfo.Unit)

	// Draw the area under the curve
	app.drawGraphArea(cr, measurements, values, times, marginLeft, marginTop,
		graphWidth, graphHeight, startTime, endTime, minVal, maxVal, metricInfo.Color)

	// Draw the line
	app.drawGraphLine(cr, measurements, values, times, marginLeft, marginTop,
		graphWidth, graphHeight, startTime, endTime, minVal, maxVal, metricInfo.Color)
}

// getMeasurementsForTimeWindow fetches measurements for the specified time window
func (app *App) getMeasurementsForTimeWindow(deviceID uint, offset time.Duration) []models.Measurement {
	endTime := time.Now().Add(offset)
	startTime := endTime.Add(-24 * time.Hour)

	var measurements []models.Measurement
	err := database.DB.Where("device_id = ? AND timestamp BETWEEN ? AND ?",
		deviceID, startTime, endTime).
		Order("timestamp ASC").
		Find(&measurements).Error
	if err != nil {
		app.logger.Error("Failed to fetch measurements for graph", "error", err)
		return nil
	}

	return measurements
}

// drawNoDataMessage displays a message when no data is available
func (app *App) drawNoDataMessage(cr *cairo.Context, width, height int) {
	cr.SetSourceRGB(0.5, 0.5, 0.5)
	cr.MoveTo(float64(width/2-50), float64(height/2))
	cr.ShowText("No data available")
}

// drawGridAndAxes draws the graph grid and axis labels
func (app *App) drawGridAndAxes(cr *cairo.Context, marginLeft, marginTop, graphWidth, graphHeight int,
	startTime, endTime time.Time, minVal, maxVal float64, unit string,
) {
	// Set grid color
	cr.SetSourceRGB(0.9, 0.9, 0.9)
	cr.SetLineWidth(1)

	// Draw horizontal grid lines (for values)
	numYLines := 5
	for i := 0; i <= numYLines; i++ {
		y := marginTop + int(float64(i)/float64(numYLines)*float64(graphHeight))
		cr.MoveTo(float64(marginLeft), float64(y))
		cr.LineTo(float64(marginLeft+graphWidth), float64(y))
		cr.Stroke()
	}

	// Draw vertical grid lines (for time)
	numXLines := 6 // Every 4 hours for 24-hour period
	for i := 0; i <= numXLines; i++ {
		x := marginLeft + int(float64(i)/float64(numXLines)*float64(graphWidth))
		cr.MoveTo(float64(x), float64(marginTop))
		cr.LineTo(float64(x), float64(marginTop+graphHeight))
		cr.Stroke()
	}

	// Draw Y-axis labels
	cr.SetSourceRGB(0.3, 0.3, 0.3)
	for i := 0; i <= numYLines; i++ {
		y := marginTop + int(float64(i)/float64(numYLines)*float64(graphHeight))
		value := maxVal - (float64(i)/float64(numYLines))*(maxVal-minVal)
		label := fmt.Sprintf("%.1f", value)
		if unit != "" {
			label += " " + unit
		}

		cr.MoveTo(5, float64(y+5))
		cr.ShowText(label)
	}

	// Draw X-axis labels (time)
	for i := 0; i <= numXLines; i++ {
		x := marginLeft + int(float64(i)/float64(numXLines)*float64(graphWidth))
		// Calculate time point: startTime + (i/numXLines) * 24 hours
		hoursFromStart := float64(i) / float64(numXLines) * 24.0
		timePoint := startTime.Add(time.Duration(hoursFromStart * float64(time.Hour)))
		label := timePoint.Format("15:04")

		cr.MoveTo(float64(x-15), float64(marginTop+graphHeight+20))
		cr.ShowText(label)
	}
}

// drawGraphArea draws the filled area under the graph line
func (app *App) drawGraphArea(cr *cairo.Context, measurements []models.Measurement, values []float64, times []time.Time,
	marginLeft, marginTop, graphWidth, graphHeight int, startTime, endTime time.Time, minVal, maxVal float64, color [3]float64,
) {
	if len(values) == 0 {
		return
	}

	// Set fill color with transparency
	cr.SetSourceRGBA(color[0], color[1], color[2], 0.3)

	// Start from bottom-left
	timeRange := endTime.Sub(startTime).Seconds()
	valueRange := maxVal - minVal

	// Move to first point
	firstTime := times[0].Sub(startTime).Seconds()
	firstX := marginLeft + int(firstTime/timeRange*float64(graphWidth))
	firstY := marginTop + int((maxVal-values[0])/valueRange*float64(graphHeight))
	cr.MoveTo(float64(firstX), float64(marginTop+graphHeight)) // Start from bottom
	cr.LineTo(float64(firstX), float64(firstY))

	// Draw line through all points
	for i := 1; i < len(values); i++ {
		timePos := times[i].Sub(startTime).Seconds()
		x := marginLeft + int(timePos/timeRange*float64(graphWidth))
		y := marginTop + int((maxVal-values[i])/valueRange*float64(graphHeight))
		cr.LineTo(float64(x), float64(y))
	}

	// Close the area back to bottom
	lastTime := times[len(times)-1].Sub(startTime).Seconds()
	lastX := marginLeft + int(lastTime/timeRange*float64(graphWidth))
	cr.LineTo(float64(lastX), float64(marginTop+graphHeight))
	cr.ClosePath()
	cr.Fill()
}

// drawGraphLine draws the graph line
func (app *App) drawGraphLine(cr *cairo.Context, measurements []models.Measurement, values []float64, times []time.Time,
	marginLeft, marginTop, graphWidth, graphHeight int, startTime, endTime time.Time, minVal, maxVal float64, color [3]float64,
) {
	if len(values) == 0 {
		return
	}

	// Set line color
	cr.SetSourceRGB(color[0], color[1], color[2])
	cr.SetLineWidth(2)

	timeRange := endTime.Sub(startTime).Seconds()
	valueRange := maxVal - minVal

	// Move to first point
	firstTime := times[0].Sub(startTime).Seconds()
	firstX := marginLeft + int(firstTime/timeRange*float64(graphWidth))
	firstY := marginTop + int((maxVal-values[0])/valueRange*float64(graphHeight))
	cr.MoveTo(float64(firstX), float64(firstY))

	// Draw line through all points
	for i := 1; i < len(values); i++ {
		timePos := times[i].Sub(startTime).Seconds()
		x := marginLeft + int(timePos/timeRange*float64(graphWidth))
		y := marginTop + int((maxVal-values[i])/valueRange*float64(graphHeight))
		cr.LineTo(float64(x), float64(y))
	}

	cr.Stroke()
}
