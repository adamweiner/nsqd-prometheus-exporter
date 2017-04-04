BINARY_NAME := nsqd-prometheus-exporter
LIST_NO_VENDOR := $(go list ./... | grep -v /vendor/)
GO_BIN := $(GOPATH)/bin
GO_VERSION := $(shell go version | awk -F ' ' '{print $$3}')
VERSION := $(shell cat VERSION)
BUILD_FLAGS := -ldflags "-X main.Version=$(VERSION)"
BUILD_PLATFORMS := darwin linux windows

default: check fmt deps test build

.PHONY: build
build:
	# Build project
	go build $(BUILD_FLAGS) -a -o $(BINARY_NAME) .

.PHONY: check
check:
	# Only continue if go is installed
	go version || ( echo "Go not installed, exiting"; exit 1 )

.PHONY: clean
clean:
	go clean -i
	rm -rf ./release
	rm -rf ./vendor/*/
	rm -f $(BINARY_NAME)

deps:
	# Install or update govend
	go get -u github.com/govend/govend
	# Fetch vendored dependencies
	$(GO_BIN)/govend -v

.PHONY: fmt
fmt:
	# Format all Go source files (excluding vendored packages)
	go fmt $(LIST_NO_VENDOR)

generate-deps:
	# Generate vendor.yml
	govend -v -l
	git checkout vendor/.gitignore

.PHONY: test
test:
	# Run all tests (excluding vendored packages)
	go test -a -v -cover $(LIST_NO_VENDOR)

.PHONY: release
release:
	# Build binaries for all BUILD_PLATFORMS
	# This is meant to be run on OSX - swap "shasum -a256" for "sha256sum -b" if running on Linux
	for platform in $(BUILD_PLATFORMS); do \
		mkdir -p release/$(BINARY_NAME)-$(VERSION).$$platform-amd64.$(GO_VERSION); \
		GOOS=$$platform GOARCH=amd64 go build $(BUILD_FLAGS) -a -o release/$(BINARY_NAME)-$(VERSION).$$platform-amd64.$(GO_VERSION)/$(BINARY_NAME) .; \
		cd release; tar -cvzf $(BINARY_NAME)-$(VERSION).$$platform-amd64.$(GO_VERSION).tar.gz $(BINARY_NAME)-$(VERSION).$$platform-amd64.$(GO_VERSION)/$(BINARY_NAME); \
		shasum -a256 $(BINARY_NAME)-$(VERSION).$$platform-amd64.$(GO_VERSION).tar.gz | awk -F ' ' '{print $$1}' >> $(BINARY_NAME)-$(VERSION).$$platform-amd64.$(GO_VERSION).sha256; \
		rm -rf $(BINARY_NAME)-$(VERSION).$$platform-amd64.$(GO_VERSION); cd ..; \
	done
