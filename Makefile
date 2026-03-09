#!/usr/bin/make -f

BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git log -1 --format='%H')
VERSION := v0.1.0-dev

BUILD_DIR ?= $(CURDIR)/build
BINARY_NAME := axond

ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=axon \
	-X github.com/cosmos/cosmos-sdk/version.AppName=$(BINARY_NAME) \
	-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
	-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT)

BUILD_FLAGS := -ldflags '$(ldflags)'

.PHONY: all build install clean test lint proto

all: build

###############################################################################
###                                Build                                    ###
###############################################################################

build:
	@echo "Building axond..."
	@go build -mod=readonly $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/axond

install:
	@echo "Installing axond..."
	@go install -mod=readonly $(BUILD_FLAGS) ./cmd/axond

clean:
	@rm -rf $(BUILD_DIR)

###############################################################################
###                               Protobuf                                  ###
###############################################################################

proto:
	@echo "Generating protobuf files..."
	@buf generate proto

proto-lint:
	@buf lint proto

###############################################################################
###                                Testing                                  ###
###############################################################################

test:
	@echo "Running tests..."
	@go test -v ./...

test-unit:
	@go test -v -count=1 ./x/...

test-cover:
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

benchmark:
	@go test -bench=. -benchmem ./...

###############################################################################
###                                Linting                                  ###
###############################################################################

lint:
	@golangci-lint run --config .golangci.yml

###############################################################################
###                             Local Testnet                               ###
###############################################################################

localnet-init:
	@echo "Initializing local testnet..."
	@$(BUILD_DIR)/$(BINARY_NAME) init test-node --chain-id axon-local-1
	@$(BUILD_DIR)/$(BINARY_NAME) genesis add-genesis-account axon1... 1000000000000000000000aaxon

localnet-start:
	@echo "Starting local node..."
	@$(BUILD_DIR)/$(BINARY_NAME) start

###############################################################################
###                              Docker                                     ###
###############################################################################

docker-build:
	@docker build -t axon-chain/axon:$(VERSION) .

docker-run:
	@docker run -it --rm -p 26656:26656 -p 26657:26657 -p 8545:8545 axon-chain/axon:$(VERSION)
