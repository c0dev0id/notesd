.PHONY: build clean test

build:
	go build -o notes-cli ./cmd/notes-cli
	go build -o notes-tui ./cmd/notes-tui

test:
	go test -v ./...

clean:
	rm -f notes-cli notes-tui
