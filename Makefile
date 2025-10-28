##
# Project Title
#
# @file
# @version 0.1

# Go Makefile

# Variables
APP=adgui
BINDIR=build
GOBUILD=go build
GOCLEAN=go clean
GOMOD=go mod

PREFIX?=/usr/local/bin

# Replace it with "sudo", "doas" or somethat, that allows root privileges on your
# system.
# SUDO=sudo
SUDO?=

# Version information
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
FLAGS := -buildvcs=false -ldflags "-X main.version=$(VERSION) -X main.gitCommit=$(COMMIT)"

.PHONY: all
all: build

.PHONY: build
build:
	$(foreach dir,$(wildcard cmd/*), $(GOBUILD) $(FLAGS) -o $(BINDIR)/ ./$(dir);)

.PHONY: release-wayland
release-wayland: # default build for Wayland
	go tool fyne build -o build/adgui-wayland --release --tags wayland ./cmd/adgui

.PHONY: release-x11
release-x11: # build for X11/XLibre
	go tool fyne build -o build/adgui-x11 --release --tags x11 ./cmd/adgui

.PHONY: test
test:
	go tool ginkgo ./...

.PHONY: run
run: build
	./$(BINDIR)/$(APP)

.PHONY: run-log
run-log: tidy build
	SLOG_LEVEL=debug ./$(BINDIR)/$(APP)

.PHONY: run-race
run-race: tidy
	go run -race $(LDFLAGS) ./cmd/$(APP)

.PHONY: lint
lint:
	go tool golangci-lint run ./...

.PHONY: vulncheck
vulncheck:
	go tool govulncheck ./...

.PHONY: tidy
tidy:
	$(GOMOD) tidy

# Build under regular user, only install under root!
.PHONY: install
install: release-x11 release-wayland
	$(SUDO) install ./build/adgui-x11 $(PREFIX)
	$(SUDO) install ./build/adgui-wayland $(PREFIX)
	$(SUDO) install ./adgui-run $(PREFIX)

.PHONY: sloc
sloc:
	cloc * >sloc.stats

.PHONY: clean
clean:
	$(GOCLEAN)
	rm -rf $(BINDIR)

# end
