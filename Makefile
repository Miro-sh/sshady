.PHONY: all build test lint clean install

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo "unknown")
LDFLAGS  = -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)
BINARY   = sshady

# Go settings
GO      ?= go
GOFLAGS ?= -trimpath

all: lint test build

build:
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) .

install:
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) .
	sudo mv $(BINARY) /usr/local/bin/

test:
	$(GO) test -v -race -coverprofile=coverage.out -covermode=atomic ./...

test-short:
	$(GO) test -short -v ./...

lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

clean:
	rm -f $(BINARY) $(BINARY).exe coverage.out coverage.html
	rm -rf dist/

# Coverage report in browser
cover: test
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Open coverage.html in your browser"

# Build for multiple platforms
dist:
	GOOS=linux   GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/sshady-linux-amd64 .
	GOOS=linux   GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/sshady-linux-arm64 .
	GOOS=darwin  GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/sshady-darwin-amd64 .
	GOOS=darwin  GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/sshady-darwin-arm64 .
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/sshady-windows-amd64.exe .
