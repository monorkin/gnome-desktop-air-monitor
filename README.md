# GNOME Desktop Air Monitor
GNOME Desktop Air Monitor is a desktop application for viewing Air Quality Monitor measurements.
It comes with a shell extension that allows you to view the measurements in the top bar.
And there is also a CLI.



https://github.com/user-attachments/assets/2e4a6b33-70c1-4e8f-9fd8-4a08c1bae1b2



Currently, the following devices are supported:
- [x] [Awair Element](https://uk.getawair.com/products/element)
- [ ] [Awair Omni](https://uk.getawair.com/products/omni)

> [!NOTE]
> Checked devices have been tested and confirmed to work with the app

## Usage

### GUI

![device index page](https://github.com/user-attachments/assets/05990759-3e0f-4d59-ac05-ee2943be4262)

![device show page](https://github.com/user-attachments/assets/c179b37f-507b-4970-97a6-2049efe422a7)

![settings page](https://github.com/user-attachments/assets/3d737ddb-ba36-42c2-b954-f0023d5197e3)

![status bar](https://github.com/user-attachments/assets/23d4371c-55fa-45d0-b753-1703aef161e1)

### CLI



## Installation

> [!IMPORTANT]
> After installation, you'll have to logout and log back into your account to load the shell extension.
> (This is required to load GNOME Shell Extensions)
> The app itself will work without logging out.
> Remember to enable the extension after logging back in!

### From a pre-compiled binary

Run the following command:

```bash
curl -sSL https://raw.githubusercontent.com/monorkin/gnome-desktop-air-monitor/main/install.sh | bash
```

This will download and install the latest version of the application for your system, including the shell extension.

To uninstall the application, you can run:

```bash
curl -sSL https://raw.githubusercontent.com/monorkin/gnome-desktop-air-monitor/main/uninstall.sh | bash
```

### Build from source

Make sure that all dependencies listed in the [development](#development) section are installed.
Then run the following commands:

```bash
make install
```

To uninstall the application, you can run:

```bash
make uninstall
```

## Development

To run the app locally, you need to have the following dependencies installed:
* Go version 1.23 or later
* GDK version 4.0 or later

To build and run the app with debug symbols, use the following command:

```bash
make dev

# To pass arguments to the CLI use the ARGS variable.
# make dev ARGS="device ls"
```

To test the shell extension, you can use the following command:

```bash
make shell-extension-dev
```

To see all available tools, run:

```bash
make help
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
