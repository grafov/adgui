# POSIX sh avoids per-$(shell) startup cost from an inherited user SHELL (e.g. fish)
# and matches recipe shells unless a target overrides SHELL.
SHELL := /bin/sh

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
release-wayland: # default build for Wayland (broken yet)
	go tool fyne build -o build/adgui-wayland --release --tags wayland ./cmd/adgui

.PHONY: release-xlibre
release-xlibre: # build for X11/XLibre
	go tool fyne build -o build/adgui-xlibre --release --tags x11 ./cmd/adgui

.PHONY: test
test:
	go test ./...

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

.PHONY: lint-fix
lint-fix:
	go tool golangci-lint run -v --fix ./...

.PHONY: vulncheck
vulncheck:
	go tool govulncheck ./...

.PHONY: tidy
tidy:
	$(GOMOD) tidy

# Build under regular user, only install under root!
.PHONY: install
install: release-xlibre release-wayland
	@echo "Don't forget to set SUDO=sudo (or SUDO=doas) before this command!"
	@echo "for example: SUDO=doas make install"
	$(SUDO) install ./build/adgui-xlibre $(PREFIX)
	$(SUDO) install ./build/adgui-wayland $(PREFIX)
	$(SUDO) install ./adgui $(PREFIX)

deploy: release-wayland release-xlibre
	go tool fyne package --target linux --exe build/adgui-wayland --icon ./res/Icon.png --release --tags wayland

# Pack portable Linux amd64 archive for GitHub Releases.
DIST_NAME=adgui-$(VERSION)-linux-amd64
DIST_DIR=$(BINDIR)/$(DIST_NAME)
DIST_ARCHIVE=$(BINDIR)/$(DIST_NAME).tar.xz

.PHONY: dist
dist: release-xlibre release-wayland
	rm -rf $(DIST_DIR)
	mkdir -p $(DIST_DIR)
	cp build/adgui-xlibre build/adgui-wayland adgui LICENSE README.md $(DIST_DIR)/
	chmod +x $(DIST_DIR)/adgui-xlibre $(DIST_DIR)/adgui-wayland $(DIST_DIR)/adgui
	tar -C $(BINDIR) -cJf $(DIST_ARCHIVE) $(DIST_NAME)
	cd $(BINDIR) && sha256sum $(DIST_NAME).tar.xz > $(DIST_NAME).tar.xz.sha256

.PHONY: sloc
sloc:
	cloc * >sloc.stats

.PHONY: clean
clean:
	$(GOCLEAN)
	rm -rf $(BINDIR)

# end
