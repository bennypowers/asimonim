.PHONY: all test lint clean install
.PHONY: linux-x64 linux-arm64 darwin-x64 darwin-arm64 win32-x64 win32-arm64

BINARY_NAME := asimonim
DIST_DIR := dist/bin
GO_BUILD_FLAGS := -ldflags="-s -w"

# Workaround for Gentoo Linux "hole in findfunctab" error with race detector
# See: https://bugs.gentoo.org/961618
ifeq ($(shell test -f /etc/gentoo-release && echo yes),yes)
    RACE_LDFLAGS := -ldflags="-linkmode=external"
else
    RACE_LDFLAGS :=
endif

all:
	go build -o $(DIST_DIR)/$(BINARY_NAME) .

install: all
	cp $(DIST_DIR)/$(BINARY_NAME) ~/.local/bin/$(BINARY_NAME)

clean:
	rm -rf $(DIST_DIR)/
	go clean -cache -testcache

test:
	go test -race $(RACE_LDFLAGS) ./...

lint:
	go vet ./...

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
