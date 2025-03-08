TIMESTAMP := $(shell date +%Y%m%d-%H%M%S)
VERSION := 1.1.0
BUILD_DATE := $(shell date +%Y-%m-%d)
LDFLAGS := -ldflags "-X main.AppVersion=$(VERSION) -X main.AppBuild=$(BUILD_DATE)"
ADDITIONAL_BUILD_FLAGS=""

ifeq ($(CI_RUN), true)
	ADDITIONAL_BUILD_FLAGS="-test.short"
endif

.PHONY: help
help:  ## display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: run
run: build ## run application
	./interruption-tracker

.PHONY: build
build: ## build the binary
	go build $(LDFLAGS) -o interruption-tracker *.go

.PHONY: test
test: ## run tests
	go test ./... -v $(ADDITIONAL_BUILD_FLAGS)

.PHONY: install
install: ## install dependencies
	go install ./...

.PHONY: release
release: test ## build release version
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o build/interruption-tracker-mac-amd64 *.go
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o build/interruption-tracker-mac-arm64 *.go
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o build/interruption-tracker-linux-amd64 *.go
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o build/interruption-tracker-windows-amd64.exe *.go

.PHONY: backup
backup: ## create a backup of data
	mkdir -p backups
	./interruption-tracker --backup=backups/backup-$(TIMESTAMP).json

.PHONY: format
format: ## format code
	go fmt ./...

.PHONY: lint
lint: ## run linters
	golangci-lint run ./...

.PHONY: clean
clean: ## clean build artifacts
	rm -f interruption-tracker
	rm -rf build/
	go clean

.PHONY: docs
docs: ## generate documentation
	mkdir -p docs
	go doc -all > docs/api-documentation.txt

.PHONY: dev
dev: build ## run with development settings
	./interruption-tracker --dev

.PHONY: stats
stats: build ## show statistics for current week
	./interruption-tracker --stats=week
