.PHONY: build clean test

BINDIR ?= .

build:
	go build -o $(BINDIR)/notes-cli ./cmd/notes-cli
	go build -o $(BINDIR)/notes-tui ./cmd/notes-tui

test:
	go test -v ./...

clean:
	rm -f $(BINDIR)/notes-cli $(BINDIR)/notes-tui
