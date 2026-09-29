.PHONY: help assets verify-assets clean build build-for build-all test version

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
MARKED_VERSION := 17.0.5
MD_MERMAIDJS_VERSION := 11.4.0
HIGHLIGHTJS_VERSION := 11.11.1
LUCIDE_VERSION := 0.469.0

# Directories
MD_STATIC_DIR := internal/generics/static
MD_JS_DIR := $(MD_STATIC_DIR)/js
MD_CSS_DIR := $(MD_STATIC_DIR)/css
MD_FONTS_DIR := $(MD_STATIC_DIR)/fonts
STAMP := $(MD_STATIC_DIR)/.assets-stamp

# Google Fonts User-Agent (returns direct woff2 files)
UA := Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36

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
assets: $(STAMP) ## Download static assets for markdown command
	@:

$(STAMP): $(MAKEFILE_LIST)
	@echo "$(CYAN)Downloading markdown viewer assets...$(NC)"
	@mkdir -p $(MD_JS_DIR) $(MD_CSS_DIR) $(MD_FONTS_DIR)
	@curl -sfL -o $(MD_JS_DIR)/marked.min.js "https://cdn.jsdelivr.net/npm/marked@$(MARKED_VERSION)/lib/marked.umd.js"
	@curl -sfL -o $(MD_JS_DIR)/mermaid.min.js "https://cdn.jsdelivr.net/npm/mermaid@$(MD_MERMAIDJS_VERSION)/dist/mermaid.min.js"
	@curl -sfL -o $(MD_JS_DIR)/highlight.min.js "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/$(HIGHLIGHTJS_VERSION)/highlight.min.js"
	@curl -sfL -o $(MD_JS_DIR)/tailwindcss.js "https://cdn.tailwindcss.com/3.4.16"
	@curl -sfL -o $(MD_JS_DIR)/lucide.min.js "https://unpkg.com/lucide@$(LUCIDE_VERSION)/dist/umd/lucide.min.js"
	@curl -sfL -o $(MD_CSS_DIR)/github-dark.min.css "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/$(HIGHLIGHTJS_VERSION)/styles/github-dark.min.css"
	@curl -sfL -H "User-Agent: $(UA)" \
		-o /tmp/inter.raw "https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap"
	@awk '/^\/\* /{keep = ($$0 ~ /^\/\* latin(-ext)? \*\/$$/)} keep' \
		/tmp/inter.raw > /tmp/inter.css
	@grep -o 'https://fonts.gstatic.com/[^)]*' /tmp/inter.css | while read url; do \
		filename=$$(basename "$$url"); \
		curl -sfL -o $(MD_FONTS_DIR)/"$$filename" "$$url"; \
	done
	@sed -E 's|https://fonts\.gstatic\.com/[^)]*/([^/)]*)|../fonts/\1|g' /tmp/inter.css > $(MD_CSS_DIR)/inter.css
	@curl -sfL -H "User-Agent: $(UA)" \
		-o /tmp/jetbrains-mono.raw "https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&display=swap"
	@awk '/^\/\* /{keep = ($$0 ~ /^\/\* latin(-ext)? \*\/$$/)} keep' \
		/tmp/jetbrains-mono.raw > /tmp/jetbrains-mono.css
	@grep -o 'https://fonts.gstatic.com/[^)]*' /tmp/jetbrains-mono.css | while read url; do \
		filename=$$(basename "$$url"); \
		curl -sfL -o $(MD_FONTS_DIR)/"$$filename" "$$url"; \
	done
	@sed -E 's|https://fonts\.gstatic\.com/[^)]*/([^/)]*)|../fonts/\1|g' /tmp/jetbrains-mono.css > $(MD_CSS_DIR)/jetbrains-mono.css
	@rm -f /tmp/inter.raw /tmp/inter.css /tmp/jetbrains-mono.raw /tmp/jetbrains-mono.css
	@touch $(STAMP)
	@echo "$(GREEN)Markdown viewer assets downloaded to $(MD_STATIC_DIR)/$(NC)"

verify-assets: ## Verify required assets exist
	@MISSING=0; \
	for f in $(MD_JS_DIR)/tailwindcss.js $(MD_JS_DIR)/marked.min.js $(MD_JS_DIR)/mermaid.min.js $(MD_JS_DIR)/highlight.min.js $(MD_JS_DIR)/lucide.min.js $(MD_CSS_DIR)/github-dark.min.css $(MD_CSS_DIR)/inter.css $(MD_CSS_DIR)/jetbrains-mono.css; do \
		if [ ! -f "$$f" ]; then \
			echo "$(YELLOW)Missing:$(NC) $$f"; \
			MISSING=1; \
		fi; \
	done; \
	if [ "$$MISSING" = "1" ]; then \
		echo "$(YELLOW)Run 'make assets' to download missing markdown viewer assets$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Assets verified$(NC)"

clean: ## Remove built artifacts and downloaded assets
	@rm -f $(APP_NAME) $(APP_NAME)-* $(STAMP)
	@find $(MD_JS_DIR) $(MD_CSS_DIR) $(MD_FONTS_DIR) -mindepth 1 ! -name '.gitkeep' -delete 2>/dev/null || true
	@echo "$(GREEN)Cleaned$(NC)"

# =============================================================================
# Build
# =============================================================================
build: assets verify-assets ## Build binary for current platform
	@go build -ldflags="-s -w -X 'github.com/tanq16/$(APP_NAME)/cmd.AppVersion=$(VERSION)'" -o $(APP_NAME) .
	@echo "$(GREEN)Built: ./$(APP_NAME)$(NC)"

build-for: verify-assets ## Build binary for specified GOOS/GOARCH
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="-s -w -X 'github.com/tanq16/$(APP_NAME)/cmd.AppVersion=$(VERSION)'" -o $(APP_NAME)-$(GOOS)-$(GOARCH) .
	@echo "$(GREEN)Built: ./$(APP_NAME)-$(GOOS)-$(GOARCH)$(NC)"

build-all: assets verify-assets ## Build all platform binaries
	@$(MAKE) build-for GOOS=linux GOARCH=amd64
	@$(MAKE) build-for GOOS=linux GOARCH=arm64
	@$(MAKE) build-for GOOS=darwin GOARCH=amd64
	@$(MAKE) build-for GOOS=darwin GOARCH=arm64

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
