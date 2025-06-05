package main

import (
	"os"

	"github.com/monorkin/gnome-desktop-air-monitor/internal/app"
)

func main() {
	app := app.NewApp()
	os.Exit(app.Run())
}
