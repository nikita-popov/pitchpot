BINARY_DIR := bin
CMD_CONFIGURATOR := ./cmd/pitchpot-configurator
CMD_SERVER := ./cmd/pitchpotd

.PHONY: all build clean test lint

all: build

build:
	go build -o $(BINARY_DIR)/pitchpot-configurator $(CMD_CONFIGURATOR)
	go build -o $(BINARY_DIR)/pitchpotd $(CMD_SERVER)

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BINARY_DIR)
