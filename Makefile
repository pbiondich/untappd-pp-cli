.PHONY: build test lint install clean

BIN_EXT := $(if $(filter windows,$(shell go env GOOS)),.exe,)

build:
	go build -o bin/untappd-pp-cli$(BIN_EXT) ./cmd/untappd-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/untappd-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/untappd-pp-mcp$(BIN_EXT) ./cmd/untappd-pp-mcp

install-mcp:
	go install ./cmd/untappd-pp-mcp

build-all: build build-mcp
