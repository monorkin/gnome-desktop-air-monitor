package app

import (
	adw "github.com/diamondburned/gotk4-adwaita/pkg/adw"
	gio "github.com/diamondburned/gotk4/pkg/gio/v2"
	gtk "github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type App struct {
	*gtk.Application
}

func NewApp() *App {
	application := gtk.NewApplication(
		"io.stanko.awair-gnome-client",
		gio.ApplicationFlagsNone,
	)

	app := &App{
		Application: application,
	}

	app.ConnectActivate(app.onActivate)

	return app
}

func (app *App) onActivate() {
	win := adw.NewApplicationWindow(app.Application)
	win.SetDefaultSize(800, 600)
	win.SetTitle("Awair Client")

	content := gtk.NewBox(gtk.OrientationVertical, 0)
	label := gtk.NewLabel("Awair Client")
	content.Append(label)

	win.SetContent(content)
	win.Present()
}

func (app *App) Run() int {
	return app.Application.Run(nil)
}
