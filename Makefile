BINARY := md-viewer
# Use system pkg-config (linuxbrew's shadows it and misses system libs)
export PKG_CONFIG := /usr/bin/pkg-config
# Shim webkit2gtk-4.0 → 4.1 for webview_go compatibility
export PKG_CONFIG_PATH := $(CURDIR)/.pkg-config:$(PKG_CONFIG_PATH)

PREFIX ?= /usr/local

.PHONY: build clean install uninstall

build:
	go build -o $(BINARY) .

clean:
	rm -f $(BINARY)

install: build
	install -Dm755 $(BINARY) $(PREFIX)/bin/$(BINARY)

uninstall:
	rm -f $(PREFIX)/bin/$(BINARY)
