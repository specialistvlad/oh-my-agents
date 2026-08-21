SHELL := /bin/bash
.DEFAULT_GOAL := help

.PHONY: install
install: ## Install dependencies
	@npm install

.PHONY: start
start: ## Run the dev server with hot reload
	@npm run start

.PHONY: dev
dev: start ## Alias for start

.PHONY: check
check: ## Build + prettier + lint + tests
	@npm run check

.PHONY: check-fix
check-fix: ## Same as check, but rewrites formatting and auto-fixable lint
	@npm run check:fix

.PHONY: test
test: ## Run unit tests
	@npm run test

.PHONY: lint
lint: ## Run eslint
	@npm run lint

.PHONY: build
build: ## Production build (vite build + tsc)
	@npm run build

.PHONY: help
help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
