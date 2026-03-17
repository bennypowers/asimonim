.PHONY: all test test-coverage lint clean install release patch minor major
.PHONY: linux-x64 linux-arm64 darwin-x64 darwin-arm64 win32-x64 win32-arm64

BINARY_NAME := asimonim
DIST_DIR := dist/bin
GO_BUILD_FLAGS := -ldflags="$(shell scripts/ldflags.sh)"

# Extract version from goals if present (e.g., "make release v0.1.0" or "make release patch")
VERSION ?= $(filter v% patch minor major,$(MAKECMDGOALS))

# Optional path to release notes file for gh release create
RELEASE_NOTES ?=

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

COVERPKGS := $(shell go list ./... | grep -v '/testutil' | grep -v '/internal/mapfs' | grep -v '/internal/logger' | grep -v '/internal/version' | grep -v '/lsp/test/' | grep -vx 'bennypowers.dev/asimonim' | paste -sd, -)
test-coverage:
	go test -race $(RACE_LDFLAGS) -coverprofile=coverage.out -coverpkg=$(COVERPKGS) ./...

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
	@RELEASE_NOTES="$(RELEASE_NOTES)" ./scripts/release.sh $(VERSION)

# Prevent make from treating version args as file targets
patch minor major:
	@:

# Catch version tags like v0.1.0
v%:
	@:

# Shared Windows cross-compilation image (from go-release-workflows)
SHARED_WINDOWS_CC_IMAGE := asimonim-shared-windows-cc

# Cross-compilation targets (CGO_ENABLED=1 required for tree-sitter)
linux-x64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
		go build $(GO_BUILD_FLAGS) \
		-o $(DIST_DIR)/$(BINARY_NAME)-linux-x64 .

linux-arm64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
		CC=aarch64-linux-gnu-gcc \
		go build $(GO_BUILD_FLAGS) \
		-o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 .

darwin-x64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
		CC="clang -arch x86_64" \
		CGO_CFLAGS="-arch x86_64" CGO_LDFLAGS="-arch x86_64" \
		go build $(GO_BUILD_FLAGS) \
		-o $(DIST_DIR)/$(BINARY_NAME)-darwin-x64 .

darwin-arm64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
		CC="clang -arch arm64" \
		CGO_CFLAGS="-arch arm64" CGO_LDFLAGS="-arch arm64" \
		go build $(GO_BUILD_FLAGS) \
		-o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 .

build-shared-windows-image:
	@if ! podman image exists $(SHARED_WINDOWS_CC_IMAGE); then \
		echo "Building shared Windows cross-compilation image..."; \
		curl -fsSL https://raw.githubusercontent.com/bennypowers/go-release-workflows/main/.github/actions/setup-windows-build/Containerfile \
			| podman build -t $(SHARED_WINDOWS_CC_IMAGE) -f - .; \
	else \
		echo "Image $(SHARED_WINDOWS_CC_IMAGE) already exists, skipping build."; \
	fi

win32-x64: build-shared-windows-image
	@mkdir -p $(DIST_DIR)
	podman run --rm \
		-v $(PWD):/src:Z \
		-w /src \
		-e GOOS=windows \
		-e GOARCH=amd64 \
		-e CGO_ENABLED=1 \
		-e CC=x86_64-w64-mingw32-gcc \
		-e CXX=x86_64-w64-mingw32-g++ \
		$(SHARED_WINDOWS_CC_IMAGE) \
		go build $(GO_BUILD_FLAGS) \
			-o $(DIST_DIR)/$(BINARY_NAME)-win32-x64.exe .

win32-arm64: build-shared-windows-image
	@mkdir -p $(DIST_DIR)
	podman run --rm \
		-v $(PWD):/src:Z \
		-w /src \
		-e GOOS=windows \
		-e GOARCH=arm64 \
		-e CGO_ENABLED=1 \
		-e CC=aarch64-w64-mingw32-gcc \
		-e CXX=aarch64-w64-mingw32-g++ \
		$(SHARED_WINDOWS_CC_IMAGE) \
		go build $(GO_BUILD_FLAGS) \
			-o $(DIST_DIR)/$(BINARY_NAME)-win32-arm64.exe .
