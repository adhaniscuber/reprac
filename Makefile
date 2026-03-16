BINARY := reprac
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_FLAGS := -ldflags="-s -w -X github.com/adhaniscuber/reprac/cmd.Version=$(VERSION)"

.PHONY: all build install run tidy clean

all: build

## build: Build the binary to ./bin/reprac
build:
	@mkdir -p bin
	go build $(BUILD_FLAGS) -o bin/$(BINARY) .

## install: Install to $GOPATH/bin (available anywhere in PATH)
install:
	go install $(BUILD_FLAGS) .

## run: Run directly with go run
run:
	go run . --config repos.yaml

## tidy: Download and tidy dependencies
tidy:
	go mod tidy

## clean: Remove build artifacts
clean:
	rm -rf bin/

## deps: Show dependency status
deps:
	go mod download
	go list -m all
