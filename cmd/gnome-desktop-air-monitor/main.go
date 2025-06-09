package main

import (
	"os"

	"github.com/monorkin/gnome-desktop-air-monitor/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
