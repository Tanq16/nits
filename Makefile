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
MARKED_VERSION := 17.0.5
MD_MERMAIDJS_VERSION := 11.4.0
HIGHLIGHTJS_VERSION := 11.11.1
LUCIDE_VERSION := 0.469.0

# Directories
STATIC_DIR := internal/server/static
JS_DIR := $(STATIC_DIR)/js
MD_STATIC_DIR := internal/generics/static
MD_JS_DIR := $(MD_STATIC_DIR)/js
MD_CSS_DIR := $(MD_STATIC_DIR)/css
MD_FONTS_DIR := $(MD_STATIC_DIR)/fonts

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
assets: ## Download static assets for mermaid-svg and markdown commands
	@echo "$(CYAN)Downloading assets...$(NC)"
	@mkdir -p $(JS_DIR)
	@curl -sL "https://cdn.tailwindcss.com" -o "$(JS_DIR)/tailwindcss.js"
	@curl -sL "https://cdn.jsdelivr.net/npm/mermaid@$(MERMAIDJS_VERSION)/dist/mermaid.min.js" -o "$(JS_DIR)/mermaid.min.js"
	@echo "$(GREEN)Assets downloaded$(NC)"
	@echo "$(CYAN)Downloading markdown viewer assets...$(NC)"
	@mkdir -p $(MD_JS_DIR) $(MD_CSS_DIR) $(MD_FONTS_DIR)
	@curl -sL -o $(MD_JS_DIR)/marked.min.js "https://cdn.jsdelivr.net/npm/marked@$(MARKED_VERSION)/lib/marked.umd.js"
	@curl -sL -o $(MD_JS_DIR)/mermaid.min.js "https://cdn.jsdelivr.net/npm/mermaid@$(MD_MERMAIDJS_VERSION)/dist/mermaid.min.js"
	@curl -sL -o $(MD_JS_DIR)/highlight.min.js "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/$(HIGHLIGHTJS_VERSION)/highlight.min.js"
	@curl -sL -o $(MD_JS_DIR)/tailwindcss.js "https://cdn.tailwindcss.com/3.4.16"
	@curl -sL -o $(MD_JS_DIR)/lucide.min.js "https://unpkg.com/lucide@$(LUCIDE_VERSION)/dist/umd/lucide.min.js"
	@curl -sL -o $(MD_CSS_DIR)/github-dark.min.css "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/$(HIGHLIGHTJS_VERSION)/styles/github-dark.min.css"
	@curl -sL -A "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36" \
		-o /tmp/inter.css "https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap"
	@grep -o 'https://[^)]*' /tmp/inter.css | while read url; do \
		filename=$$(basename "$$url"); \
		curl -sL -o $(MD_FONTS_DIR)/"$$filename" "$$url"; \
	done
	@sed -E 's|url\(https://[^)]*/([^/)]*)\)|url(../fonts/\1)|g' /tmp/inter.css > $(MD_CSS_DIR)/inter.css
	@curl -sL -A "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36" \
		-o /tmp/jetbrains-mono.css "https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&display=swap"
	@grep -o 'https://[^)]*' /tmp/jetbrains-mono.css | while read url; do \
		filename=$$(basename "$$url"); \
		curl -sL -o $(MD_FONTS_DIR)/"$$filename" "$$url"; \
	done
	@sed -E 's|url\(https://[^)]*/([^/)]*)\)|url(../fonts/\1)|g' /tmp/jetbrains-mono.css > $(MD_CSS_DIR)/jetbrains-mono.css
	@rm -f /tmp/inter.css /tmp/jetbrains-mono.css
	@echo "$(GREEN)Markdown viewer assets downloaded to $(MD_STATIC_DIR)/$(NC)"

verify-assets: ## Verify required assets exist
	@test -f $(JS_DIR)/tailwindcss.js || (echo "$(YELLOW)tailwindcss.js missing. Run 'make assets'$(NC)" && exit 1)
	@test -f $(JS_DIR)/mermaid.min.js || (echo "$(YELLOW)mermaid.min.js missing. Run 'make assets'$(NC)" && exit 1)
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
	@rm -f $(APP_NAME) $(APP_NAME)-*
	@rm -rf $(JS_DIR)/*.js
	@rm -rf $(MD_JS_DIR) $(MD_CSS_DIR) $(MD_FONTS_DIR)
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
