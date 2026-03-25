.PHONY: build test clean run

BINDIR ?= .

build:
	go build -o $(BINDIR)/notesd ./cmd/notesd

test:
	go test -v ./...

clean:
	rm -f $(BINDIR)/notesd

run: build
	$(BINDIR)/notesd
