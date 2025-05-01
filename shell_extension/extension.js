// extension.js
'use strict';

const { Clutter, GObject, St } = imports.gi;
const Main = imports.ui.main;
const PanelMenu = imports.ui.panelMenu;
const PopupMenu = imports.ui.popupMenu;
const ByteArray = imports.byteArray;
const Gio = imports.gi.Gio;

// DBus interface info
const AwairDbusInterface = `<node>
  <interface name="com.example.AwairMonitor">
    <method name="GetDevices">
      <arg type="aa{sv}" direction="out" name="devices"/>
    </method>
    <method name="RefreshDevices">
    </method>
    <method name="GetPinnedMetrics">
      <arg type="as" direction="out" name="metrics"/>
    </method>
    <method name="SetPinnedMetric">
      <arg type="s" direction="in" name="metric"/>
      <arg type="b" direction="in" name="pinned"/>
    </method>
    <signal name="DevicesUpdated">
      <arg type="aa{sv}" name="devices"/>
    </signal>
    <signal name="PinnedMetricsChanged">
      <arg type="as" name="metrics"/>
    </signal>
  </interface>
</node>`;

const AwairProxy = Gio.DBusProxy.makeProxyWrapper(AwairDbusInterface);

// Awair indicator component
var AwairIndicator = GObject.registerClass(
class AwairIndicator extends PanelMenu.Button {
    _init() {
        super._init(0.0, 'Awair Indicator');
        
        // Create the tray icon
        this._icon = new St.Icon({
            icon_name: 'sensors-applet',
            style_class: 'system-status-icon',
        });
        
        // Primary display (shows main pinned metric)
        this._primaryIndicator = new St.Label({
            text: '...',
            y_align: Clutter.ActorAlign.CENTER,
        });
        
        // Box to hold icon and text
        let box = new St.BoxLayout();
        box.add_child(this._icon);
        box.add_child(this._primaryIndicator);
        this.add_child(box);
        
        // Initialize menu
        this._initMenu();
        
        // Connect to DBus
        this._connectDBus();
        
        // Refresh data every 60 seconds
        this._refreshTimeout = null;
        this._startRefreshTimer();
    }
    
    _initMenu() {
        // Header section
        this._headerSection = new PopupMenu.PopupMenuSection();
        this.menu.addMenuItem(this._headerSection);
        
        // Device sections will be added dynamically
        this._deviceSections = {};
        
        // Add separator
        this.menu.addMenuItem(new PopupMenu.PopupSeparatorMenuItem());
        
        // Settings section
        let settingsSection = new PopupMenu.PopupMenuSection();
        
        // Pin metrics options
        this._pinMetricsMenu = new PopupMenu.PopupSubMenuMenuItem('Pin Metrics');
        this._initPinMetricsMenu();
        settingsSection.addMenuItem(this._pinMetricsMenu);
        
        // Open main app button
        let openAppItem = new PopupMenu.PopupMenuItem('Open Awair Monitor');
        openAppItem.connect('activate', () => {
            let appInfo = Gio.AppInfo.create_from_commandline(
                'awair-monitor', 'Awair Monitor', 
                Gio.AppInfoCreateFlags.NONE
            );
            appInfo.launch([], null);
        });
        settingsSection.addMenuItem(openAppItem);
        
        // Refresh now button
        let refreshItem = new PopupMenu.PopupMenuItem('Refresh Now');
        refreshItem.connect('activate', () => {
            this._refreshData(true);
        });
        settingsSection.addMenuItem(refreshItem);
        
        this.menu.addMenuItem(settingsSection);
    }
    
    _initPinMetricsMenu() {
        // Available metrics
        const metrics = [
            { id: 'temp', name: 'Temperature', unit: '°C' },
            { id: 'humidity', name: 'Humidity', unit: '%' },
            { id: 'co2', name: 'CO₂', unit: 'ppm' },
            { id: 'voc', name: 'VOC', unit: 'ppb' },
            { id: 'pm25', name: 'PM2.5', unit: 'μg/m³' },
            { id: 'score', name: 'Air Score', unit: '' }
        ];
        
        // Clear existing items
        this._pinMetricsMenu.menu.removeAll();
        
        // Add metric options
        this._metricSwitches = {};
        metrics.forEach(metric => {
            let item = new PopupMenu.PopupSwitchMenuItem(
                `${metric.name} ${metric.unit}`, 
                false
            );
            item.connect('toggled', (item, state) => {
                if (this._proxy) {
                    this._proxy.SetPinnedMetricRemote(metric.id, state);
                }
            });
            this._metricSwitches[metric.id] = item;
            this._pinMetricsMenu.menu.addMenuItem(item);
        });
    }
    
    _connectDBus() {
        try {
            this._proxy = new AwairProxy(
                Gio.DBus.session,
                'com.example.AwairMonitor',
                '/com/example/AwairMonitor'
            );
            
            // Connect signals
            this._proxy.connectSignal('DevicesUpdated', 
                (proxy, nameOwner, [devices]) => {
                    this._updateDevices(devices);
                }
            );
            
            this._proxy.connectSignal('PinnedMetricsChanged',
                (proxy, nameOwner, [metrics]) => {
                    this._updatePinnedMetrics(metrics);
                }
            );
            
            // Initial data fetch
            this._refreshData(true);
        } catch (e) {
            logError(e, 'Failed to connect to Awair Monitor DBus service');
            this._showError('Failed to connect to Awair Monitor service');
        }
    }
    
    _startRefreshTimer() {
        if (this._refreshTimeout) {
            GLib.source_remove(this._refreshTimeout);
        }
        
        this._refreshTimeout = GLib.timeout_add_seconds(
            GLib.PRIORITY_DEFAULT,
            60, // 60 seconds
            () => {
                this._refreshData(false);
                return GLib.SOURCE_CONTINUE;
            }
        );
    }
    
    _refreshData(forceRefresh) {
        if (!this._proxy) {
            return;
        }
        
        // Get pinned metrics
        this._proxy.GetPinnedMetricsRemote((result, error) => {
            if (error) {
                logError(error, 'Failed to get pinned metrics');
                return;
            }
            
            let [metrics] = result;
            this._updatePinnedMetrics(metrics);
        });
        
        // Get devices
        this._proxy.GetDevicesRemote((result, error) => {
            if (error) {
                logError(error, 'Failed to get devices');
                return;
            }
            
            let [devices] = result;
            this._updateDevices(devices);
        });
        
        // Request a refresh if needed
        if (forceRefresh) {
            this._proxy.RefreshDevicesRemote((result, error) => {
                if (error) {
                    logError(error, 'Failed to refresh devices');
                }
            });
        }
    }
    
    _updatePinnedMetrics(metrics) {
        // Update pin menu switches
        for (let metricId in this._metricSwitches) {
            let isPinned = metrics.includes(metricId);
            let item = this._metricSwitches[metricId];
            if (item.state !== isPinned) {
                item.setToggleState(isPinned);
            }
        }
        
        // Update primary display
        if (metrics.length > 0) {
            // Use the first pinned metric as primary indicator
            this._primaryMetric = metrics[0];
        } else {
            this._primaryMetric = 'score'; // Default
        }
        
        // Update display
        this._updateDisplay();
    }
    
    _updateDevices(devices) {
        // Clear existing device sections
        for (let deviceId in this._deviceSections) {
            if (this._deviceSections[deviceId]) {
                this._deviceSections[deviceId].destroy();
                delete this._deviceSections[deviceId];
            }
        }
        
        // Update header
        this._headerSection.removeAll();
        if (devices.length === 0) {
            let item = new PopupMenu.PopupMenuItem('No devices found');
            item.setSensitive(false);
            this._headerSection.addMenuItem(item);
        } else {
            let item = new PopupMenu.PopupMenuItem(`${devices.length} Awair devices found`);
            item.setSensitive(false);
            this._headerSection.addMenuItem(item);
        }
        
        // Add device sections
        devices.forEach(device => {
            let deviceId = device.DeviceID.unpack();
            let deviceType = device.Type.unpack();
            let deviceName = `${deviceType} (${deviceId})`;
            
            // Create device section
            let section = new PopupMenu.PopupMenuSection();
            let title = new PopupMenu.PopupMenuItem(deviceName);
            title.setSensitive(false);
            section.addMenuItem(title);
            
            // Add metrics
            this._addMetricItem(section, 'Temperature', 
                device.Temperature?.unpack() || 'N/A', '°C');
            this._addMetricItem(section, 'Humidity', 
                device.Humidity?.unpack() || 'N/A', '%');
            this._addMetricItem(section, 'CO₂', 
                device.CO2?.unpack() || 'N/A', 'ppm');
            this._addMetricItem(section, 'VOC', 
                device.VOC?.unpack() || 'N/A', 'ppb');
            this._addMetricItem(section, 'PM2.5', 
                device.PM25?.unpack() || 'N/A', 'μg/m³');
            this._addMetricItem(section, 'Air Score', 
                device.Score?.unpack() || 'N/A', '');
            
            // Add to menu
            this.menu.addMenuItem(section);
            this._deviceSections[deviceId] = section;
        });
        
        // Store devices
        this._devices = devices;
        
        // Update display
        this._updateDisplay();
    }
    
    _addMetricItem(section, name, value, unit) {
        let text = `${name}: ${value}${unit}`;
        let item = new PopupMenu.PopupMenuItem(text);
        item.setSensitive(false);
        section.addMenuItem(item);
    }
    
    _updateDisplay() {
        if (!this._devices || this._devices.length === 0) {
            this._primaryIndicator.text = '...';
            return;
        }
        
        // Get average value for the primary metric
        let total = 0;
        let count = 0;
        let unit = '';
        
        switch (this._primaryMetric) {
            case 'temp':
                this._devices.forEach(d => {
                    if (d.Temperature) {
                        total += d.Temperature.unpack();
                        count++;
                    }
                });
                unit = '°C';
                break;
            case 'humidity':
                this._devices.forEach(d => {
                    if (d.Humidity) {
                        total += d.Humidity.unpack();
                        count++;
                    }
                });
                unit = '%';
                break;
            case 'co2':
                this._devices.forEach(d => {
                    if (d.CO2) {
                        total += d.CO2.unpack();
                        count++;
                    }
                });
                unit = 'ppm';
                break;
            case 'voc':
                this._devices.forEach(d => {
                    if (d.VOC) {
                        total += d.VOC.unpack();
                        count++;
                    }
                });
                unit = 'ppb';
                break;
            case 'pm25':
                this._devices.forEach(d => {
                    if (d.PM25) {
                        total += d.PM25.unpack();
                        count++;
                    }
                });
                unit = 'μg/m³';
                break;
            case 'score':
            default:
                this._devices.forEach(d => {
                    if (d.Score) {
                        total += d.Score.unpack();
                        count++;
                    }
                });
                unit = '';
                break;
        }
        
        if (count > 0) {
            let value = (total / count).toFixed(1);
            this._primaryIndicator.text = `${value}${unit}`;
        } else {
            this._primaryIndicator.text = '...';
        }
    }
    
    _showError(message) {
        this._primaryIndicator.text = '!';
        
        this._headerSection.removeAll();
        let item = new PopupMenu.PopupMenuItem(message);
        item.setSensitive(false);
        this._headerSection.addMenuItem(item);
    }
    
    destroy() {
        if (this._refreshTimeout) {
            GLib.source_remove(this._refreshTimeout);
            this._refreshTimeout = null;
        }
        
        super.destroy();
    }
});

// Extension hooks
class Extension {
    constructor() {
        this._indicator = null;
    }
    
    enable() {
        this._indicator = new AwairIndicator();
        Main.panel.addToStatusArea('awair-indicator', this._indicator);
    }
    
    disable() {
        if (this._indicator !== null) {
            this._indicator.destroy();
            this._indicator = null;
        }
    }
}

function init() {
    return new Extension();
}
