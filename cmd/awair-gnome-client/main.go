package main

import (
	"os"

	"github.com/monorkin/awair-gnome-client/internal/app"
)

func main() {
	app := app.NewApp()
	os.Exit(app.Run())
}
