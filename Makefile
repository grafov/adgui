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

# Version information
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.gitCommit=$(COMMIT)"

.PHONY: all
all: build

.PHONY: build
build:
	$(foreach dir,$(wildcard cmd/*), $(GOBUILD) $(LDFLAGS) -o $(BINDIR)/ ./$(dir);)

.PHONY: test
test:
	go tool ginkgo ./...

.PHONY: clean
clean:
	$(GOCLEAN)
	rm -rf $(BINDIR)

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

.PHONY: tidy
tidy:
	$(GOMOD) tidy

.PHONY: sloc
sloc:
	cloc * >sloc.stats

# end
