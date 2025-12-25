ROOT_DIR := $(shell pwd)
BIN_DIR := $(ROOT_DIR)/bin
GO := go

AUTH_DIR := auth-server
SYNC_DIR := sync-backend/cmd/server

all: build

build: auth sync

auth:
	@echo "Building auth-server"
	@mkdir -p $(BIN_DIR)
	cd $(AUTH_DIR) && $(GO) build -trimpath -o $(BIN_DIR)/auth-server

sync:
	@echo "Building sync-backend"
	@mkdir -p $(BIN_DIR)
	cd $(SYNC_DIR) && $(GO) build -trimpath -o $(BIN_DIR)/sync-backend