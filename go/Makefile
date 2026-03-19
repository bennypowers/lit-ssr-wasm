DIST ?= dist/bin
LDFLAGS := -ldflags '-s -w'

.PHONY: all clean linux-x64 linux-arm64 darwin-x64 darwin-arm64 win32-x64 win32-arm64

all: linux-x64 linux-arm64 darwin-x64 darwin-arm64 win32-x64 win32-arm64

clean:
	rm -rf $(DIST)

define BUILD
	CGO_ENABLED=0 GOOS=$(1) GOARCH=$(2) go build $(LDFLAGS) -o $(DIST)/lit-ssr-builtin-$(3) ./cmd/lit-ssr
	CGO_ENABLED=0 GOOS=$(1) GOARCH=$(2) go build $(LDFLAGS) -o $(DIST)/lit-ssr-runtime-$(3) ./cmd/lit-ssr-runtime
endef

linux-x64:
	$(call BUILD,linux,amd64,linux-x64)

linux-arm64:
	$(call BUILD,linux,arm64,linux-arm64)

darwin-x64:
	$(call BUILD,darwin,amd64,darwin-x64)

darwin-arm64:
	$(call BUILD,darwin,arm64,darwin-arm64)

win32-x64:
	$(call BUILD,windows,amd64,win32-x64.exe)

win32-arm64:
	$(call BUILD,windows,arm64,win32-arm64.exe)
