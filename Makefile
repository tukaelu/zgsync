PROG_NAME ?= zgsync
VERSION ?= $(shell git describe --tags --abbrev=0)
REVISION := $(shell git rev-parse --short HEAD)

OUTPUT_DIR ?= dist

OS := $(shell go env GOOS)
ARCH := $(shell go env GOARCH)
BUILD_LDFLAGS := "-X github.com/tukaelu/zgsync.Version=$(VERSION) -X github.com/tukaelu/zgsync.Revision=$(REVISION)"

SOURCES = $(shell find . -type f -name '*.go')

.PHONY: test
test:
	go test -v ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: build
build: $(OUTPUT_DIR) $(SOURCES)
	go mod tidy
	GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 go build -ldflags=$(BUILD_LDFLAGS) -o $(OUTPUT_DIR)/$(PROG_NAME) ./cmd/$(PROG_NAME)/

.PHONY: clean
clean:
	rm -r $(OUTPUT_DIR)/*

$(OUTPUT_DIR):
	mkdir -p $(OUTPUT_DIR)