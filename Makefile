APP_NAME := northstar-ai
ENTRYPOINT := ./src/cmd/northstar
DIST_DIR := dist
TAG ?= latest
VERSIONED_NAME := $(APP_NAME)-$(TAG)
RELEASE_DIR := $(DIST_DIR)/$(TAG)

.PHONY: help build-release clean

help:
	@printf "Targets available:\n"
	@printf "  make build-release TAG=v1.0.0\n"
	@printf "  make clean\n"

build-release:
	@mkdir -p $(RELEASE_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(RELEASE_DIR)/$(VERSIONED_NAME)-linux-amd64 $(ENTRYPOINT)
	GOOS=linux GOARCH=arm64 go build -o $(RELEASE_DIR)/$(VERSIONED_NAME)-linux-arm64 $(ENTRYPOINT)
	GOOS=windows GOARCH=amd64 go build -o $(RELEASE_DIR)/$(VERSIONED_NAME)-windows-amd64.exe $(ENTRYPOINT)
	GOOS=windows GOARCH=arm64 go build -o $(RELEASE_DIR)/$(VERSIONED_NAME)-windows-arm64.exe $(ENTRYPOINT)
	GOOS=darwin GOARCH=amd64 go build -o $(RELEASE_DIR)/$(VERSIONED_NAME)-darwin-amd64 $(ENTRYPOINT)
	GOOS=darwin GOARCH=arm64 go build -o $(RELEASE_DIR)/$(VERSIONED_NAME)-darwin-arm64 $(ENTRYPOINT)

build-local:
	go build -o northstar ./src/cmd/northstar

clean:
	rm -rf $(DIST_DIR)
