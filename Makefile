# Simple Makefile to build for Linux and Windows

APP_NAME := notion-row-deleter
DIST := dist
PKG := ./
LDFLAGS := -s -w
BUILD_FLAGS := -trimpath -ldflags "$(LDFLAGS)"

# Default target
.PHONY: help
help:
	@echo "Targets:"
	@echo "  run                   Run locally"
	@echo "  build                 Build for host platform"
	@echo "  build-linux           Build Linux (amd64, arm64)"
	@echo "  build-windows         Build Windows (amd64, arm64)"
	@echo "  build-all             Build Linux & Windows (amd64, arm64)"
	@echo "  clean                 Remove dist directory"

$(DIST):
	@mkdir -p $(DIST)

.PHONY: run
run:
	go run .

.PHONY: build
build: $(DIST)
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(DIST)/$(APP_NAME) $(PKG)

# Linux builds
.PHONY: build-linux build-linux-amd64 build-linux-arm64
build-linux: build-linux-amd64 build-linux-arm64

build-linux-amd64: $(DIST)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(DIST)/$(APP_NAME)-linux-amd64 $(PKG)

build-linux-arm64: $(DIST)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(DIST)/$(APP_NAME)-linux-arm64 $(PKG)

# Windows builds
.PHONY: build-windows build-windows-amd64 build-windows-arm64
build-windows: build-windows-amd64 build-windows-arm64

build-windows-amd64: $(DIST)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(DIST)/$(APP_NAME)-windows-amd64.exe $(PKG)

build-windows-arm64: $(DIST)
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(DIST)/$(APP_NAME)-windows-arm64.exe $(PKG)

# All targets
.PHONY: build-all
build-all: build-linux build-windows

.PHONY: clean
clean:
	rm -rf $(DIST)
