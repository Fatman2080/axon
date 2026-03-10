#!/usr/bin/make -f

BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git log -1 --format='%H')
VERSION := v0.3.0-dev

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
	@echo "Running all tests..."
	@go test -v -count=1 ./x/agent/...

test-unit:
	@echo "Running unit tests..."
	@go test -v -count=1 -run "Test" ./x/agent/keeper/

test-cover:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out -covermode=atomic ./x/agent/keeper/
	@go tool cover -func=coverage.out | tail -1
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-economics:
	@echo "Running economics tests..."
	@go test -v -run "TestBlockReward|TestContribution|TestMaxShare|TestDeflation" ./x/agent/keeper/

test-agent:
	@echo "Running agent module tests..."
	@go test -v -run "TestDefaultParams|TestChallengePool|TestScoreResponse|TestKeyFunctions" ./x/agent/keeper/

benchmark:
	@go test -bench=. -benchmem ./x/agent/...

###############################################################################
###                                Linting                                  ###
###############################################################################

lint:
	@golangci-lint run --config .golangci.yml

###############################################################################
###                             Local Testnet                               ###
###############################################################################

localnet-init:
	@echo "Initializing single-node testnet..."
	@bash scripts/local_node.sh

localnet-start:
	@echo "Starting local node..."
	@$(BUILD_DIR)/$(BINARY_NAME) start --home $$HOME/.axond --chain-id axon_9001-1 --json-rpc.enable

localnet-4node:
	@echo "Setting up 4-node localnet..."
	@bash scripts/localnet.sh

localnet-4node-start:
	@bash $$HOME/.axon-localnet/start_all.sh

localnet-4node-stop:
	@bash $$HOME/.axon-localnet/stop_all.sh

###############################################################################
###                              Docker                                     ###
###############################################################################

docker-build:
	@docker build -t axon-chain/axon:$(VERSION) .

docker-run:
	@docker run -it --rm -p 26656:26656 -p 26657:26657 -p 8545:8545 axon-chain/axon:$(VERSION)

###############################################################################
###                          Public Testnet                                 ###
###############################################################################

testnet-up:
	@echo "Starting Axon public testnet (Docker)..."
	@docker compose -f testnet/docker-compose.yml up -d --build
	@echo ""
	@echo "  JSON-RPC: http://localhost:8545"
	@echo "  Faucet:   http://localhost:8080"
	@echo "  Explorer: http://localhost:4000"
	@echo "  CometBFT: http://localhost:26657"

testnet-down:
	@docker compose -f testnet/docker-compose.yml down

testnet-reset:
	@docker compose -f testnet/docker-compose.yml down -v
	@echo "Testnet data cleared."

testnet-logs:
	@docker compose -f testnet/docker-compose.yml logs -f

testnet-status:
	@docker compose -f testnet/docker-compose.yml ps

monitoring-up:
	@docker compose -f testnet/monitoring/docker-compose.yml up -d
	@echo "  Grafana:    http://localhost:3000 (admin/axon)"
	@echo "  Prometheus: http://localhost:9091"

monitoring-down:
	@docker compose -f testnet/monitoring/docker-compose.yml down
