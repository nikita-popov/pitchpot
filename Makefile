GO       = go
BIN_DIR := bin
CMD_CONFIGURATOR := ./cmd/pitchpot-configurator
CMD_SERVER := ./cmd/pitchpotd
VERSION := $(shell printf '%s-dev' "$$(git describe --tags --always --dirty 2>/dev/null || echo unknown)")
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build clean test lint

all: deps build

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/pitchpot-configurator $(CMD_CONFIGURATOR)
	$(GO) build -o $(BIN_DIR)/pitchpotd $(CMD_SERVER)

deps:
	$(GO) mod tidy
	$(GO) mod download

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

test:
	$(GO) test -v -race ./...

check: fmt vet test

clean:
	rm -rf $(BIN_DIR)
