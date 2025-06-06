import { Extension } from "resource:///org/gnome/shell/extensions/extension.js";
import * as Main from "resource:///org/gnome/shell/ui/main.js";
import * as PanelMenu from "resource:///org/gnome/shell/ui/panelMenu.js";
import * as PopupMenu from "resource:///org/gnome/shell/ui/popupMenu.js";

const { GObject, St, Gio, GLib, Clutter } = imports.gi;

// DBUS interface for Air Monitor communication
const AirMonitorInterface = `<node>
  <interface name="io.stanko.AirMonitor">
    <method name="GetSelectedDevice">
      <arg type="a{sv}" direction="out" name="device"/>
    </method>
    <method name="OpenApp">
    </method>
    <method name="OpenSettings">
    </method>
    <method name="Quit">
    </method>
    <signal name="DeviceUpdated">
      <arg type="a{sv}" name="device"/>
    </signal>
  </interface>
</node>`;

const AirMonitorProxy = Gio.DBusProxy.makeProxyWrapper(AirMonitorInterface);

const AirMonitorIndicator = GObject.registerClass(
  class AirMonitorIndicator extends PanelMenu.Button {
    _init() {
      super._init(0.0, "Air Monitor", false);

      // Create the icon and score display
      this._icon = new St.Icon({
        icon_name: "weather-clear-symbolic",
        style_class: "system-status-icon",
      });

      this._scoreLabel = new St.Label({
        text: "--",
        y_align: Clutter.ActorAlign.CENTER,
        style: "margin-left: 4px; font-weight: bold;",
      });

      // Container for icon and score
      const box = new St.BoxLayout({
        style_class: "panel-status-menu-box",
      });
      box.add_child(this._icon);
      box.add_child(this._scoreLabel);
      this.add_child(box);

      // Initialize menu
      this._buildMenu();

      // Connect to DBUS
      this._connectDBus();

      // Start periodic updates
      this._startUpdateTimer();

      // Current device data
      this._currentDevice = null;
    }

    _buildMenu() {
      // Device info section
      this._deviceSection = new PopupMenu.PopupMenuSection();
      this.menu.addMenuItem(this._deviceSection);

      // Separator
      this.menu.addMenuItem(new PopupMenu.PopupSeparatorMenuItem());

      // Action buttons
      const actionsSection = new PopupMenu.PopupMenuSection();

      // Open app button
      this._openAppItem = new PopupMenu.PopupMenuItem("Open Air Monitor");
      this._openAppItem.connect("activate", () => this._openApp());
      actionsSection.addMenuItem(this._openAppItem);

      // Settings button
      this._settingsItem = new PopupMenu.PopupMenuItem("Settings");
      this._settingsItem.connect("activate", () => this._openSettings());
      actionsSection.addMenuItem(this._settingsItem);

      // Separator before quit
      actionsSection.addMenuItem(new PopupMenu.PopupSeparatorMenuItem());

      // Quit button
      this._quitItem = new PopupMenu.PopupMenuItem("Quit Air Monitor");
      this._quitItem.connect("activate", () => this._quitApp());
      actionsSection.addMenuItem(this._quitItem);

      this.menu.addMenuItem(actionsSection);

      // Initial state
      this._updateDeviceDisplay(null);
    }

    _connectDBus() {
      try {
        this._proxy = new AirMonitorProxy(
          Gio.DBus.session,
          "io.stanko.AirMonitor",
          "/io/stanko/AirMonitor",
        );

        // Connect to device update signals
        this._proxy.connectSignal(
          "DeviceUpdated",
          (proxy, nameOwner, [deviceData]) => {
            this._updateDeviceDisplay(deviceData);
          },
        );

        // Monitor service availability
        this._nameWatcherId = Gio.DBus.session.watch_name(
          "io.stanko.AirMonitor",
          Gio.BusNameWatcherFlags.NONE,
          () => {
            // Service appeared
            this.visible = true;
            this._refreshDeviceData();
          },
          () => {
            // Service disappeared
            this.visible = false;
            this._showError("Service not available");
          }
        );

        // Get initial device data
        this._refreshDeviceData();
      } catch (e) {
        console.error("Failed to connect to Air Monitor DBUS service:", e);
        this._showError("Service not available");
      }
    }

    _startUpdateTimer() {
      // Update every 30 seconds
      this._updateTimeout = GLib.timeout_add_seconds(
        GLib.PRIORITY_DEFAULT,
        30,
        () => {
          this._refreshDeviceData();
          return GLib.SOURCE_CONTINUE;
        },
      );
    }

    _refreshDeviceData() {
      if (!this._proxy) {
        return;
      }

      try {
        this._proxy.GetSelectedDeviceRemote((result, error) => {
          if (error) {
            console.error("Failed to get device data:", error);
            this._showError("Connection error");
            return;
          }

          const [deviceData] = result;
          this._updateDeviceDisplay(deviceData);
        });
      } catch (e) {
        console.error("Error calling GetSelectedDevice:", e);
        this._showError("Service error");
      }
    }

    _updateDeviceDisplay(deviceData) {
      this._currentDevice = deviceData;

      if (!deviceData) {
        // No device selected or service not running
        this._scoreLabel.text = "--";
        this._icon.icon_name = "weather-clear-symbolic";
        this._updateDeviceMenu("No device selected", []);
        return;
      }

      // Update score in tray
      const score = deviceData.score?.unpack() || 0;
      this._scoreLabel.text = Math.round(score).toString();

      // Update icon color based on score
      if (score < 30) {
        this._icon.icon_name = "weather-severe-alert-symbolic";
        this._scoreLabel.style =
          "margin-left: 4px; font-weight: bold; color: #e74c3c;";
      } else if (score < 75) {
        this._icon.icon_name = "weather-overcast-symbolic";
        this._scoreLabel.style =
          "margin-left: 4px; font-weight: bold; color: #f39c12;";
      } else {
        this._icon.icon_name = "weather-clear-symbolic";
        this._scoreLabel.style =
          "margin-left: 4px; font-weight: bold; color: #27ae60;";
      }

      // Prepare measurements for menu
      const measurements = [
        { label: "Air Quality Score", value: score.toFixed(0), unit: "" },
        {
          label: "Temperature",
          value: deviceData.temperature?.unpack()?.toFixed(1) || "--",
          unit: "°C",
        },
        {
          label: "Humidity",
          value: deviceData.humidity?.unpack()?.toFixed(1) || "--",
          unit: "%",
        },
        {
          label: "CO₂",
          value: deviceData.co2?.unpack()?.toFixed(0) || "--",
          unit: " ppm",
        },
        {
          label: "VOC",
          value: deviceData.voc?.unpack()?.toFixed(0) || "--",
          unit: " ppb",
        },
        {
          label: "PM2.5",
          value: deviceData.pm25?.unpack()?.toFixed(1) || "--",
          unit: " μg/m³",
        },
      ];

      const deviceName = deviceData.name?.unpack() || "Unknown Device";
      this._updateDeviceMenu(deviceName, measurements);
    }

    _updateDeviceMenu(deviceName, measurements) {
      // Clear existing device section
      this._deviceSection.removeAll();

      // Device name header
      const deviceHeader = new PopupMenu.PopupMenuItem(deviceName);
      deviceHeader.setSensitive(false);
      deviceHeader.label.style = "font-weight: bold;";
      this._deviceSection.addMenuItem(deviceHeader);

      // Add measurements
      measurements.forEach((measurement) => {
        const text = `${measurement.label}: ${measurement.value}${measurement.unit}`;
        const item = new PopupMenu.PopupMenuItem(text);
        item.setSensitive(false);
        item.label.style = "padding-left: 20px;";
        this._deviceSection.addMenuItem(item);
      });

      // If no measurements, show message
      if (measurements.length === 0) {
        const noDataItem = new PopupMenu.PopupMenuItem(
          "No measurements available",
        );
        noDataItem.setSensitive(false);
        noDataItem.label.style = "padding-left: 20px; font-style: italic;";
        this._deviceSection.addMenuItem(noDataItem);
      }
    }

    _showError(message) {
      this._scoreLabel.text = "!";
      this._icon.icon_name = "dialog-error-symbolic";
      this._scoreLabel.style =
        "margin-left: 4px; font-weight: bold; color: #e74c3c;";
      this._updateDeviceMenu(message, []);
    }

    _openApp() {
      if (this._proxy) {
        try {
          this._proxy.OpenAppRemote((result, error) => {
            if (error) {
              console.error("Failed to open app:", error);
            }
          });
        } catch (e) {
          console.error("Error calling OpenApp:", e);
        }
      }
    }

    _openSettings() {
      if (this._proxy) {
        try {
          this._proxy.OpenSettingsRemote((result, error) => {
            if (error) {
              console.error("Failed to open settings:", error);
            }
          });
        } catch (e) {
          console.error("Error calling OpenSettings:", e);
        }
      }
    }

    _quitApp() {
      if (this._proxy) {
        try {
          this._proxy.QuitRemote((result, error) => {
            if (error) {
              console.error("Failed to quit app:", error);
            }
          });
        } catch (e) {
          console.error("Error calling Quit:", e);
        }
      }
    }

    destroy() {
      if (this._updateTimeout) {
        GLib.source_remove(this._updateTimeout);
        this._updateTimeout = null;
      }

      if (this._nameWatcherId) {
        Gio.DBus.session.unwatch_name(this._nameWatcherId);
        this._nameWatcherId = null;
      }

      super.destroy();
    }
  },
);

export default class AirMonitorExtension extends Extension {
  constructor(metadata) {
    super(metadata);
    this._indicator = null;
  }

  enable() {
    this._indicator = new AirMonitorIndicator();
    Main.panel.addToStatusArea(this.uuid, this._indicator);
  }

  disable() {
    if (this._indicator) {
      this._indicator.destroy();
      this._indicator = null;
    }
  }
}
