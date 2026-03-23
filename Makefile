.PHONY: build test test-coverage e2e clean clean-all install uninstall lint fmt check security build-all release-snapshot install-tools clean-tools dev-setup help demo print-versions compare-tool-versions update-tool-versions

PREFIX ?= /usr/local
DESTDIR ?=

# Build the kira binary
build:
	$(eval GIT_TAG := $(shell git describe --tags --always 2>/dev/null || echo dev))
	$(eval GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown))
	$(eval BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ))
	$(eval GIT_DIRTY := $(shell test -n "$(shell git status --porcelain 2>/dev/null)" && echo dirty || echo clean))
	go build -ldflags "-X 'kira/internal/commands.Version=$(GIT_TAG)' -X 'kira/internal/commands.Commit=$(GIT_COMMIT)' -X 'kira/internal/commands.BuildDate=$(BUILD_DATE)' -X 'kira/internal/commands.Dirty=$(GIT_DIRTY)'" -o kira cmd/kira/main.go

# Run tests (use "make check verbose" or "make test verbose" for verbose)
test:
	go test $(if $(filter verbose,$(MAKECMDGOALS)),-v,) ./...

# Dummy target so "make check verbose" passes -v to go test
verbose:
	@true

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run end-to-end tests
e2e:
	bash kira_e2e_tests.sh $(ARGS)

# Clean build artifacts
clean:
	rm -f kira coverage.out coverage.html kira-*-amd64 kira-*-amd64.exe
	rm -rf dist/

# Clean everything: build artifacts, dev tools, and installed binary
clean-all: clean clean-tools uninstall
	@echo "All cleanup complete"

# Install kira to $(PREFIX)/bin
install: build
	@ERROR_OUTPUT=$$(install -d "$(DESTDIR)$(PREFIX)/bin" 2>&1); \
	EXIT_CODE=$$?; \
	if [ $$EXIT_CODE -ne 0 ]; then \
		if echo "$$ERROR_OUTPUT" | grep -qi "permission\|denied"; then \
			echo "Error: Permission denied. Try running with sudo:"; \
			echo "  sudo make install"; \
			exit 1; \
		else \
			echo "$$ERROR_OUTPUT"; \
			exit $$EXIT_CODE; \
		fi; \
	fi
	@ERROR_OUTPUT=$$(install -m 0755 kira "$(DESTDIR)$(PREFIX)/bin/kira" 2>&1); \
	EXIT_CODE=$$?; \
	if [ $$EXIT_CODE -ne 0 ]; then \
		if echo "$$ERROR_OUTPUT" | grep -qi "permission\|denied"; then \
			echo "Error: Permission denied. Try running with sudo:"; \
			echo "  sudo make install"; \
			exit 1; \
		else \
			echo "$$ERROR_OUTPUT"; \
			exit $$EXIT_CODE; \
		fi; \
	fi
	@echo "kira installed successfully to $(DESTDIR)$(PREFIX)/bin/kira"

# Uninstall kira from $(PREFIX)/bin
uninstall:
	@echo "Uninstalling kira from $(DESTDIR)$(PREFIX)/bin..."
	@rm -f "$(DESTDIR)$(PREFIX)/bin/kira"
	@echo "kira uninstalled"

# Run linter
lint:
	@PATH="$$(go env GOPATH)/bin:$$PATH" golangci-lint run

# Format code (writes changes) via golangci-lint config
fmt:
	@PATH="$$(go env GOPATH)/bin:$$PATH" golangci-lint run --fix

# Run vulnerability check (gosec runs via golangci-lint)
security:
	@echo "Running govulncheck vulnerability scanner..."
	@PATH="$$(go env GOPATH)/bin:$$PATH" govulncheck ./...

# Run all checks (lint includes formatting, vet, and gosec security checks via golangci-lint)
check: print-versions lint security test

# Print tool versions used by checks
print-versions:
	@echo "== Tool Versions =="
	@go version
	@if [ -x "$$(go env GOPATH)/bin/golangci-lint" ]; then \
		"$$(go env GOPATH)/bin/golangci-lint" version; \
	elif command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint version; \
	else \
		echo "golangci-lint: not found"; \
	fi
	@if [ -x "$$(go env GOPATH)/bin/govulncheck" ]; then \
		"$$(go env GOPATH)/bin/govulncheck" -version; \
	elif command -v govulncheck >/dev/null 2>&1; then \
		govulncheck -version; \
	else \
		echo "govulncheck: not found"; \
	fi

# Compare installed lint/security tools to latest upstream (needs network for GitHub + module proxy)
compare-tool-versions:
	@echo "== Compare tool versions (installed vs latest) =="
	@BIN=$$(go env GOPATH)/bin; \
	GL_INST="(not found)"; \
	if [ -x "$$BIN/golangci-lint" ]; then \
	  GL_INST=$$("$$BIN/golangci-lint" version --short 2>/dev/null | head -1 || echo ""); \
	elif command -v golangci-lint >/dev/null 2>&1; then \
	  GL_INST=$$(golangci-lint version --short 2>/dev/null | head -1 || echo ""); \
	fi; \
	GL_LAT=$$(curl -sSfL https://api.github.com/repos/golangci/golangci-lint/releases/latest 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || echo "(could not fetch)"); \
	echo "golangci-lint   installed: $$GL_INST"; \
	echo "golangci-lint   latest:    $$GL_LAT"; \
	GV_INST="(not found)"; \
	if [ -x "$$BIN/govulncheck" ]; then \
	  GV_INST=$$("$$BIN/govulncheck" -version 2>/dev/null | grep -oE 'govulncheck@v[0-9.]+' | head -1 | sed 's/^govulncheck@//' || echo ""); \
	elif command -v govulncheck >/dev/null 2>&1; then \
	  GV_INST=$$(govulncheck -version 2>/dev/null | grep -oE 'govulncheck@v[0-9.]+' | head -1 | sed 's/^govulncheck@//' || echo ""); \
	fi; \
	GV_LAT=$$(go list -m golang.org/x/vuln@latest 2>/dev/null | awk '{print $$2}' || echo "(could not fetch)"); \
	echo "govulncheck     installed: $$GV_INST"; \
	echo "govulncheck     latest:    $$GV_LAT"; \
	echo "go (toolchain)  $$(go version | sed 's/^go version //')"; \
	if [ -f go.mod ]; then echo "go (go.mod)      $$(grep '^go ' go.mod | head -1 | sed 's/^go //')"; fi

# Reinstall golangci-lint (GOLANGCI_LINT_VERSION) and govulncheck (@latest) into GOPATH/bin
update-tool-versions:
	@echo "Updating golangci-lint ($(GOLANGCI_LINT_VERSION)) and govulncheck (@latest)..."
	@BIN_DIR=$$(go env GOPATH)/bin; \
	INSTALL_SCRIPT=$$(mktemp) || exit 1; \
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh -o $$INSTALL_SCRIPT || (rm -f $$INSTALL_SCRIPT; exit 1); \
	sh $$INSTALL_SCRIPT -b $$BIN_DIR $(GOLANGCI_LINT_VERSION); \
	rm -f $$INSTALL_SCRIPT; \
	go install golang.org/x/vuln/cmd/govulncheck@latest; \
	echo "Done. Run: make compare-tool-versions"

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o kira-linux-amd64 cmd/kira/main.go
	GOOS=darwin GOARCH=amd64 go build -o kira-darwin-amd64 cmd/kira/main.go
	GOOS=windows GOARCH=amd64 go build -o kira-windows-amd64.exe cmd/kira/main.go

# Test release locally using GoReleaser (does not create GitHub release)
release-snapshot:
	@PATH="$$(go env GOPATH)/bin:$$PATH"; \
	if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "GoReleaser not found. Install it with: make install-tools"; \
		exit 1; \
	fi; \
	goreleaser release --snapshot --clean

# Install required developer tools
install-tools:
	@echo "Installing developer tools..."
	@BIN_DIR=$$(go env GOPATH)/bin; \
	  echo "Checking if golangci-lint is already installed..."; \
	  if command -v golangci-lint >/dev/null 2>&1; then \
	    echo "golangci-lint found, checking version..."; \
	    INSTALLED_VERSION=$$(golangci-lint version --short 2>/dev/null | head -1 || echo ""); \
	    if [ "$$INSTALLED_VERSION" = "$(GOLANGCI_LINT_VERSION)" ]; then \
	      echo "golangci-lint $(GOLANGCI_LINT_VERSION) already installed, skipping download"; \
	      exit 0; \
	    else \
	      echo "golangci-lint $$INSTALLED_VERSION installed, upgrading to $(GOLANGCI_LINT_VERSION)"; \
	    fi; \
	  fi; \
	  echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION) to $$BIN_DIR"; \
	  echo "Downloading install script to temporary file (safer than piping curl to sh)..."; \
	  INSTALL_SCRIPT=$$(mktemp) || exit 1; \
	  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh -o $$INSTALL_SCRIPT || (rm -f $$INSTALL_SCRIPT; exit 1); \
	  echo "Running install script (it handles SHA256 checksum verification)..."; \
	  sh $$INSTALL_SCRIPT -b $$BIN_DIR $(GOLANGCI_LINT_VERSION); \
	  echo "Cleaning up temporary install script..."; \
	  rm -f $$INSTALL_SCRIPT; \
	  echo ""; \
	  echo "Installing GoReleaser..."; \
	  if command -v goreleaser >/dev/null 2>&1; then \
	    echo "GoReleaser already installed, skipping"; \
	  else \
	    echo "Downloading GoReleaser binary..."; \
	    GORELEASER_VERSION=$$(curl -s https://api.github.com/repos/goreleaser/goreleaser/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || echo "latest"); \
	    ARCH=$$(uname -m); \
	    OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
	    if [ "$$ARCH" = "arm64" ] || [ "$$ARCH" = "aarch64" ]; then \
	      ARCH="arm64"; \
	    else \
	      ARCH="x86_64"; \
	    fi; \
	    if [ "$$OS" = "darwin" ]; then \
	      OS="Darwin"; \
	    elif [ "$$OS" = "linux" ]; then \
	      OS="Linux"; \
	    fi; \
	    curl -sSfL "https://github.com/goreleaser/goreleaser/releases/latest/download/goreleaser_$${OS}_$${ARCH}.tar.gz" | tar -xz -C /tmp && mv /tmp/goreleaser "$$BIN_DIR/" && chmod +x "$$BIN_DIR/goreleaser" || echo "Failed to install GoReleaser"; \
	  fi; \
	  echo ""; \
	  echo "Installing govulncheck..."; \
	  if command -v govulncheck >/dev/null 2>&1; then \
	    echo "govulncheck already installed, skipping"; \
	  else \
	    go install golang.org/x/vuln/cmd/govulncheck@latest || echo "Failed to install govulncheck"; \
	  fi

# Clean up developer tools installed by install-tools/dev-setup
clean-tools:
	@echo "Cleaning up developer tools..."
	@BIN_DIR=$$(go env GOPATH)/bin; \
	  if [ -f "$$BIN_DIR/golangci-lint" ]; then \
	    echo "Removing golangci-lint from $$BIN_DIR..."; \
	    rm -f "$$BIN_DIR/golangci-lint"; \
	    echo "golangci-lint removed"; \
	  else \
	    echo "golangci-lint not found in $$BIN_DIR"; \
	  fi; \
	  if [ -f "$$BIN_DIR/goreleaser" ]; then \
	    echo "Removing goreleaser from $$BIN_DIR..."; \
	    rm -f "$$BIN_DIR/goreleaser"; \
	    echo "goreleaser removed"; \
	  else \
	    echo "goreleaser not found in $$BIN_DIR"; \
	  fi; \
	  if [ -f "$$BIN_DIR/govulncheck" ]; then \
	    echo "Removing govulncheck from $$BIN_DIR..."; \
	    rm -f "$$BIN_DIR/govulncheck"; \
	    echo "govulncheck removed"; \
	  else \
	    echo "govulncheck not found in $$BIN_DIR"; \
	  fi
	@echo "Developer tools cleaned up"

# Development setup
GOLANGCI_LINT_VERSION ?= latest

dev-setup: install-tools
	go mod download
	go mod tidy

# Demo initialization
demo:
	./kira init demo-workspace
	cd demo-workspace && ../kira new prd "Demo Feature" todo "This is a demo feature"
	cd demo-workspace && ../kira move 001 doing
	cd demo-workspace && ../kira save "Initial demo setup"

