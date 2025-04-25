.PHONY: build run clean

build:
	go build -o bin/awair-gnome-client ./cmd/awair-gnome-client

run: build
	./bin/awair-gnome-client

clean:
	rm -rf bin/
