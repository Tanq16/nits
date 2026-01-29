.PHONY: help assets verify-assets clean build-local build build-all test version

# =============================================================================
# Variables
# =============================================================================
APP_NAME := nits
DOCKER_USER := tanq16

# Build variables (set by CI or use defaults)
VERSION ?= dev-build
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Asset versions
TAILWIND_VERSION := latest
MERMAIDJS_VERSION := 10.9.0

# Directories
STATIC_DIR := internal/mermaidsvg/static
JS_DIR := $(STATIC_DIR)/js

# Console colors
CYAN := \033[0;36m
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m

# =============================================================================
# Help
# =============================================================================
help: ## Show this help
	@echo "$(CYAN)Available targets:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

.DEFAULT_GOAL := help

# =============================================================================
# Assets
# =============================================================================
assets: ## Download static assets for mermaid-svg command
	@echo "$(CYAN)Downloading assets...$(NC)"
	@mkdir -p $(JS_DIR)
	@curl -sL "https://cdn.tailwindcss.com" -o "$(JS_DIR)/tailwindcss.js"
	@curl -sL "https://cdn.jsdelivr.net/npm/mermaid@$(MERMAIDJS_VERSION)/dist/mermaid.min.js" -o "$(JS_DIR)/mermaid.min.js"
	@echo "$(GREEN)Assets downloaded$(NC)"

verify-assets: ## Verify required assets exist
	@test -f $(JS_DIR)/tailwindcss.js || (echo "$(YELLOW)tailwindcss.js missing. Run 'make assets'$(NC)" && exit 1)
	@test -f $(JS_DIR)/mermaid.min.js || (echo "$(YELLOW)mermaid.min.js missing. Run 'make assets'$(NC)" && exit 1)
	@echo "$(GREEN)Assets verified$(NC)"

clean: ## Remove built artifacts and downloaded assets
	@rm -f $(APP_NAME) $(APP_NAME)-*
	@rm -rf $(JS_DIR)/*.js
	@echo "$(GREEN)Cleaned$(NC)"

# =============================================================================
# Build
# =============================================================================
build-local: assets verify-assets ## Build binary for current platform
	@go build -ldflags="-s -w -X 'github.com/tanq16/$(APP_NAME)/cmd.AppVersion=$(VERSION)'" -o $(APP_NAME) .
	@echo "$(GREEN)Built: ./$(APP_NAME)$(NC)"

build: verify-assets ## Build binary for specified GOOS/GOARCH
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="-s -w -X 'github.com/tanq16/$(APP_NAME)/cmd.AppVersion=$(VERSION)'" -o $(APP_NAME)-$(GOOS)-$(GOARCH) .
	@echo "$(GREEN)Built: ./$(APP_NAME)-$(GOOS)-$(GOARCH)$(NC)"

build-all: assets verify-assets ## Build all platform binaries
	@$(MAKE) build GOOS=linux GOARCH=amd64
	@$(MAKE) build GOOS=linux GOARCH=arm64
	@$(MAKE) build GOOS=darwin GOARCH=amd64
	@$(MAKE) build GOOS=darwin GOARCH=arm64

# =============================================================================
# Test
# =============================================================================
test: ## Run tests
	@go test -v ./...

# =============================================================================
# Version
# =============================================================================
version: ## Calculate next version from commit message
	@LATEST_TAG=$$(git tag --sort=-v:refname | head -n1 || echo "0.0.0"); \
	LATEST_TAG=$${LATEST_TAG#v}; \
	MAJOR=$$(echo "$$LATEST_TAG" | cut -d. -f1); \
	MINOR=$$(echo "$$LATEST_TAG" | cut -d. -f2); \
	PATCH=$$(echo "$$LATEST_TAG" | cut -d. -f3); \
	MAJOR=$${MAJOR:-0}; MINOR=$${MINOR:-0}; PATCH=$${PATCH:-0}; \
	COMMIT_MSG="$$(git log -1 --pretty=%B)"; \
	if echo "$$COMMIT_MSG" | grep -q "\[major-release\]"; then \
		MAJOR=$$((MAJOR + 1)); MINOR=0; PATCH=0; \
	elif echo "$$COMMIT_MSG" | grep -q "\[minor-release\]"; then \
		MINOR=$$((MINOR + 1)); PATCH=0; \
	else \
		PATCH=$$((PATCH + 1)); \
	fi; \
	echo "v$${MAJOR}.$${MINOR}.$${PATCH}"
