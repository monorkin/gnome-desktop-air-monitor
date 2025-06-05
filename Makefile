.PHONY: build run clean

build:
	go build -o bin/gnome-desktop-air-monitor ./cmd/gnome-desktop-air-monitor

run: build
	./bin/gnome-desktop-air-monitor

clean:
	rm -rf bin/
