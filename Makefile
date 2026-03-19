BINARY_NAME := lit-ssr
DIST := dist/bin

.PHONY: all clean linux-x64 linux-arm64 darwin-x64 darwin-arm64 win32-x64 win32-arm64

all: linux-x64 linux-arm64 darwin-x64 darwin-arm64 win32-x64 win32-arm64

clean:
	rm -rf $(DIST)

linux-x64 linux-arm64 darwin-x64 darwin-arm64 win32-x64 win32-arm64:
	$(MAKE) -C go $@ DIST=$(CURDIR)/$(DIST)
