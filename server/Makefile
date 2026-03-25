.PHONY: build test clean run

build:
	go build -o notesd ./cmd/notesd

test:
	go test -v ./...

clean:
	rm -f notesd

run: build
	./notesd
