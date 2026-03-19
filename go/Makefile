BINARY_NAME := lit-ssr
DIST ?= dist/bin

.PHONY: all clean linux-x64 linux-arm64 darwin-x64 darwin-arm64 win32-x64 win32-arm64

all: linux-x64 linux-arm64 darwin-x64 darwin-arm64 win32-x64 win32-arm64

clean:
	rm -rf $(DIST)

linux-x64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(DIST)/$(BINARY_NAME)-linux-x64 ./cmd/lit-ssr

linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o $(DIST)/$(BINARY_NAME)-linux-arm64 ./cmd/lit-ssr

darwin-x64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $(DIST)/$(BINARY_NAME)-darwin-x64 ./cmd/lit-ssr

darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o $(DIST)/$(BINARY_NAME)-darwin-arm64 ./cmd/lit-ssr

win32-x64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $(DIST)/$(BINARY_NAME)-win32-x64.exe ./cmd/lit-ssr

win32-arm64:
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -o $(DIST)/$(BINARY_NAME)-win32-arm64.exe ./cmd/lit-ssr
