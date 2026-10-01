# Pikapu developer tasks. The Makefile is the canonical entry point for
# validation; Docker-based targets are the default way to test (see AGENTS.md).
# Run `make help` for a summary.

GO ?= go
NPM ?= npm
NODE ?= node
DOCKER ?= docker
DOCKER_BUILD ?= $(DOCKER) build
DOCKER_BUILD_ARGS ?=
DOCKER_IMAGE ?= pikapu:dev
GO_IMAGE ?= golang:1.26
NODE_IMAGE ?= node:24-alpine
PIKAPU_SMOKE_PORT ?= 17660

# Git Bash on Windows rewrites arguments that look like POSIX paths (such as
# `-w /src`) before they reach docker.exe. These variables disable that and
# are ignored on other platforms.
export MSYS_NO_PATHCONV := 1
export MSYS2_ARG_CONV_EXCL := *
export PIKAPU_SMOKE_PORT

ifeq ($(OS),Windows_NT)
APP_VERSION := $(shell powershell -NoProfile -Command "(Get-Content -Raw VERSION).Trim()")
else
APP_VERSION := $(shell tr -d '\r\n' < VERSION)
endif
LDFLAGS := -X pikapu/internal/buildinfo.Version=$(APP_VERSION)

# Named volumes keep module and build caches between containerized runs.
GO_CACHE_MOUNTS := -v pikapu-gomod:/go/pkg/mod -v pikapu-gobuild:/root/.cache/go-build
# The anonymous volume keeps Linux node_modules out of the host checkout.
NODE_MOUNTS := -v "$(CURDIR)/frontend:/src" -v /src/node_modules -v pikapu-npm:/root/.npm

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show available targets
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-22s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ---------------------------------------------------------------------------
# Docker-based validation (default)
# ---------------------------------------------------------------------------

.PHONY: test test-backend test-frontend smoke ci
test: test-backend test-frontend smoke ## Run the full containerized test suite

test-backend: ## Backend format check, vet and race-enabled tests in a Go container
	$(DOCKER) run --rm -v "$(CURDIR)/backend:/src" -w /src $(GO_CACHE_MOUNTS) $(GO_IMAGE) \
		sh -c 'test -z "$$(gofmt -l .)" || { echo "gofmt needed:"; gofmt -l .; exit 1; }; \
		go vet ./... && go test -race -count=1 ./...'

test-frontend: ## Frontend install, format check, lint, typecheck and build in a Node container
	$(DOCKER) run --rm $(NODE_MOUNTS) -w /src $(NODE_IMAGE) \
		sh -c 'npm ci --no-audit --no-fund && npm run format:check && npm run lint && \
		npm run typecheck && npx vite build --outDir /tmp/dist --emptyOutDir'

smoke: docker-build ## Build the production image and smoke test a disposable container
	$(NODE) scripts/smoke.mjs $(DOCKER_IMAGE)

ci: test ## Alias for the full containerized suite

# ---------------------------------------------------------------------------
# Local toolchain (fast iteration; not a substitute for `make test`)
# ---------------------------------------------------------------------------

.PHONY: backend-format backend-vet backend-test backend-coverage backend-build backend-run
backend-format: ## Check Go formatting with the local toolchain
	cd backend && test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

backend-vet: ## Run go vet locally
	cd backend && $(GO) vet ./...

backend-test: ## Run backend tests locally
	cd backend && $(GO) test ./...

backend-coverage: ## Run backend tests locally with a coverage summary
	cd backend && $(GO) test -count=1 -coverprofile coverage.out ./... && $(GO) tool cover -func coverage.out

backend-build: ## Build the backend binary to backend/bin (serves an embedded frontend only if built)
	cd backend && $(GO) build -ldflags "$(LDFLAGS)" -o bin/pikapu ./cmd/pikapu

# Development mode turns sign-in off for local previews; run with
# PIKAPU_MODE=production to work on the sign-in and setup flows.
backend-run: ## Run the backend locally on :7660 with data in ./data (sign-in off by default)
	cd backend && PIKAPU_DATA_DIR=../data PIKAPU_MODE=$${PIKAPU_MODE:-development} $(GO) run -ldflags "$(LDFLAGS)" ./cmd/pikapu

.PHONY: frontend-install frontend-dev frontend-lint frontend-format frontend-build
frontend-install: ## Install frontend dependencies from the lockfile
	cd frontend && $(NPM) ci --no-audit --no-fund

frontend-dev: ## Start the Vite dev server (proxies /api to :7660)
	cd frontend && $(NPM) run dev

frontend-lint: ## Lint and typecheck the frontend locally
	cd frontend && $(NPM) run lint && $(NPM) run typecheck

frontend-format: ## Check frontend formatting locally
	cd frontend && $(NPM) run format:check

frontend-build: ## Build the frontend locally into frontend/dist
	cd frontend && $(NPM) run build

# The PNG icons are committed; rerun this only when the logo changes.
ICON_PNG := docs/assets/pikapu-icon.png
ICON_MASKABLE_PNG := docs/assets/pikapu-icon-maskable.png
.PHONY: icons
icons: ## Render the app icons in frontend/public from docs/assets (container; needs network)
	$(DOCKER) run --rm -v "$(CURDIR):/src" -w /src alpine:3.22 sh -c '		apk add --no-cache -q imagemagick oxipng && 		magick $(ICON_PNG) -filter Lanczos -resize 32x32 frontend/public/favicon-32.png && 		magick $(ICON_PNG) -filter Lanczos -resize 192x192 frontend/public/icon-192.png && 		magick $(ICON_PNG) -filter Lanczos -resize 512x512 frontend/public/icon-512.png && 		magick $(ICON_MASKABLE_PNG) -filter Lanczos -resize 512x512 frontend/public/icon-maskable-512.png && 		magick $(ICON_MASKABLE_PNG) -filter Lanczos -resize 180x180 frontend/public/apple-touch-icon.png && 		oxipng -q -o 4 --strip safe docs/assets/*.png frontend/public/*.png'

# ---------------------------------------------------------------------------
# Docker runtime
# ---------------------------------------------------------------------------

.PHONY: docker-build docker-up docker-down docker-status docker-logs
docker-build: ## Build the production image as $(DOCKER_IMAGE)
	$(DOCKER_BUILD) $(DOCKER_BUILD_ARGS) -t $(DOCKER_IMAGE) .

docker-up: ## Build and start the Compose stack on port 7660
	$(DOCKER) compose up -d --build

docker-down: ## Stop the Compose stack (data in ./data is kept)
	$(DOCKER) compose down

docker-status: ## Show Compose service status
	$(DOCKER) compose ps

docker-logs: ## Show recent service logs
	$(DOCKER) compose logs --no-color --tail 200 pikapu
