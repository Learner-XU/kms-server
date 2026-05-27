APP_NAME := kms-server
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR := build
LDFLAGS := -s -w -X main.version=$(VERSION)

# Platforms: darwin/amd64 darwin/arm64 linux/amd64 linux/arm64
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

.PHONY: all clean build build-all build-linux build-darwin docker

all: build

build:
	@mkdir -p $(BUILD_DIR)
	go build -mod=vendor -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server/
	@echo "✅ Built $(BUILD_DIR)/$(APP_NAME)"

# Build for all platforms
build-all: $(PLATFORMS)

$(PLATFORMS):
	$(eval GOOS := $(word 1,$(subst /, ,$@)))
	$(eval GOARCH := $(word 2,$(subst /, ,$@)))
	@mkdir -p $(BUILD_DIR)
	@echo "📦 Building $(GOOS)/$(GOARCH)..."
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -mod=vendor -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-$(GOOS)-$(GOARCH) ./cmd/server/
	@echo "✅ $(BUILD_DIR)/$(APP_NAME)-$(GOOS)-$(GOARCH)"

build-linux:
	$(MAKE) build-all PLATFORMS="linux/amd64 linux/arm64"

build-darwin:
	$(MAKE) build-all PLATFORMS="darwin/amd64 darwin/arm64"

# Docker
docker:
	docker build -t $(APP_NAME):$(VERSION) -t $(APP_NAME):latest .

run:
	./$(BUILD_DIR)/$(APP_NAME)

clean:
	rm -rf $(BUILD_DIR)

# Show build info
info:
	@echo "App:     $(APP_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Platforms: $(PLATFORMS)"
