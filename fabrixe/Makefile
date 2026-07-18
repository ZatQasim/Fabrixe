BINARY_NAME   := fabrixe
VERSION       := 1.0.0
BUILD_DIR     := dist
WEB_DIR       := $(BUILD_DIR)/web
INSTALL_DIR   := /opt/fabrixe
DATA_DIR      := /var/lib/fabrixe
CONFIG_DIR    := /etc/fabrixe
SERVICE_FILE  := /etc/systemd/system/fabrixe.service

GO            := go
GOFLAGS       := -trimpath
LDFLAGS       := -s -w \
	-X 'main.Version=$(VERSION)' \
	-X 'main.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)'

CGO_ENABLED   ?= 1   # required for go-sqlite3

.PHONY: all backend frontend build package install uninstall clean test tidy

## Build everything (backend + frontend)
all: build

## Download Go dependencies
tidy:
	cd backend && $(GO) mod tidy && $(GO) mod verify

## Build the Go backend binary
backend:
	@echo "==> Building Fabrixe backend..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME) ./backend/cmd/fabrixe
	@echo "    Binary: $(BUILD_DIR)/$(BINARY_NAME)"

## Build the React frontend
frontend:
	@echo "==> Building Fabrixe frontend..."
	@cd frontend && npm install --legacy-peer-deps
	@cd frontend && npm run build
	@echo "    Web:    $(WEB_DIR)"

## Full build
build: backend frontend
	@echo "==> Build complete."

## Initialize Fabrixe on the local system (run as root after install)
init: $(BUILD_DIR)/$(BINARY_NAME)
	$(BUILD_DIR)/$(BINARY_NAME) -config $(CONFIG_DIR)/config.yaml -init

## Install Fabrixe system-wide (requires root)
install: build
	@echo "==> Installing Fabrixe to $(INSTALL_DIR)..."
	install -d $(INSTALL_DIR)/bin $(INSTALL_DIR)/web $(DATA_DIR) $(CONFIG_DIR)/certs
	install -m 755 $(BUILD_DIR)/$(BINARY_NAME)   $(INSTALL_DIR)/bin/$(BINARY_NAME)
	cp -r $(WEB_DIR)/.                            $(INSTALL_DIR)/web/
	if [ ! -f $(CONFIG_DIR)/config.yaml ]; then \
		install -m 640 deploy/config.yaml $(CONFIG_DIR)/config.yaml; \
	fi
	install -m 644 deploy/fabrixe.service $(SERVICE_FILE)
	ln -sf $(INSTALL_DIR)/bin/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	systemctl daemon-reload
	@echo "==> Run: systemctl enable --now fabrixe && fabrixe -init"
	@echo "    Then visit https://fabrixe.local"

## Remove Fabrixe from the system (requires root)
uninstall:
	@echo "==> Uninstalling Fabrixe..."
	-systemctl stop fabrixe
	-systemctl disable fabrixe
	rm -f $(SERVICE_FILE)
	rm -rf $(INSTALL_DIR)
	rm -f /usr/local/bin/$(BINARY_NAME)
	systemctl daemon-reload
	@echo "    Data preserved at $(DATA_DIR) and $(CONFIG_DIR)"
	@echo "    Remove manually with: rm -rf $(DATA_DIR) $(CONFIG_DIR)"

## Run tests
test:
	cd backend && CGO_ENABLED=$(CGO_ENABLED) $(GO) test ./...

## Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

## Create release tarball: Fabrixe-v<VERSION>.tar.gz
package: build
	@echo "==> Packaging Fabrixe v$(VERSION)..."
	@mkdir -p release
	@STAGING=$$(mktemp -d) && \
	PKGDIR="$$STAGING/fabrixe-$(VERSION)" && \
	mkdir -p "$$PKGDIR/bin" "$$PKGDIR/web" "$$PKGDIR/docs" && \
	cp $(BUILD_DIR)/$(BINARY_NAME) "$$PKGDIR/bin/" && \
	cp -r $(WEB_DIR)/. "$$PKGDIR/web/" && \
	cp deploy/config.yaml "$$PKGDIR/" && \
	cp deploy/install.sh "$$PKGDIR/" && \
	cp deploy/uninstall.sh "$$PKGDIR/" && \
	cp deploy/fabrixe.service "$$PKGDIR/" && \
	cp docs/installation.md "$$PKGDIR/docs/" && \
	cp docs/user-guide.md "$$PKGDIR/docs/" && \
	cp docs/developer-guide.md "$$PKGDIR/docs/" && \
	chmod +x "$$PKGDIR/install.sh" "$$PKGDIR/uninstall.sh" && \
	tar -czf release/Fabrixe-v$(VERSION).tar.gz -C "$$STAGING" fabrixe-$(VERSION) && \
	rm -rf "$$STAGING" && \
	echo "==> Package: release/Fabrixe-v$(VERSION).tar.gz" && \
	ls -lh release/Fabrixe-v$(VERSION).tar.gz

## Build release package without running install
release: package
	@echo "==> Release ready: release/Fabrixe-v$(VERSION).tar.gz"
