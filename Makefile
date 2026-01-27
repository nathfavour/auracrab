.PHONY: build run test clean

BINARY_NAME=auracrab

VERSION=$(shell git describe --tags --always || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD || echo "none")
BUILD_DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-X github.com/nathfavour/auracrab/internal/cli.Version=$(VERSION) -X github.com/nathfavour/auracrab/internal/cli.Commit=$(COMMIT) -X github.com/nathfavour/auracrab/internal/cli.BuildDate=$(BUILD_DATE)

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) cmd/$(BINARY_NAME)/main.go

run:
	go run cmd/$(BINARY_NAME)/main.go start

test:
	go test ./...

clean:
	go clean
	rm -f bin/$(BINARY_NAME)
