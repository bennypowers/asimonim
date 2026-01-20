.PHONY: all test lint clean install release patch minor major
.PHONY: linux-x64 linux-arm64 darwin-x64 darwin-arm64 win32-x64 win32-arm64

BINARY_NAME := asimonim
DIST_DIR := dist/bin
GO_BUILD_FLAGS := -ldflags="$(shell scripts/ldflags.sh)"

# Extract version from goals if present (e.g., "make release v0.1.0" or "make release patch")
VERSION ?= $(filter v% patch minor major,$(MAKECMDGOALS))

# Workaround for Gentoo Linux "hole in findfunctab" error with race detector
# See: https://bugs.gentoo.org/961618
ifeq ($(shell test -f /etc/gentoo-release && echo yes),yes)
    RACE_LDFLAGS := -ldflags="-linkmode=external"
else
    RACE_LDFLAGS :=
endif

all:
	@mkdir -p $(DIST_DIR)
	go build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME) .

install: all
	cp $(DIST_DIR)/$(BINARY_NAME) ~/.local/bin/$(BINARY_NAME)

clean:
	rm -rf $(DIST_DIR)/
	go clean -cache -testcache

test:
	go test -race $(RACE_LDFLAGS) ./...

lint:
	go vet ./...

release:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION or bump type is required"; \
		echo "Usage: make release <version|patch|minor|major>"; \
		echo "  make release v0.1.0  - Release explicit version"; \
		echo "  make release patch   - Bump patch version (0.0.x)"; \
		echo "  make release minor   - Bump minor version (0.x.0)"; \
		echo "  make release major   - Bump major version (x.0.0)"; \
		exit 1; \
	fi
	@./scripts/release.sh $(VERSION)

# Prevent make from treating version args as file targets
patch minor major:
	@:

# Catch version tags like v0.1.0
v%:
	@:

# Cross-compilation targets (CGO_ENABLED=0 for pure Go)
linux-x64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-x64 .

linux-arm64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 .

darwin-x64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-x64 .

darwin-arm64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 .

win32-x64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-win32-x64.exe .

win32-arm64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-win32-arm64.exe .
