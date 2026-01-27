.PHONY: build run test clean

BINARY_NAME=auracrab

build:
	go build -o bin/$(BINARY_NAME) cmd/$(BINARY_NAME)/main.go

run:
	go run cmd/$(BINARY_NAME)/main.go

test:
	go test ./...

clean:
	go clean
	rm -f bin/$(BINARY_NAME)
