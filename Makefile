API := services/api
WEB := services/web

SHELL := /bin/bash
.DEFAULT_GOAL := help

# ─── Setup ────────────────────────────────────────────────────────────────────

.PHONY: setup
setup: ## Install toolchains and dependencies for both services
	@$(MAKE) -C $(API) setup
	@$(MAKE) -C $(API) install
	@cd $(WEB) && npm install
	@[ -f .env ] || cp .env.example .env

# ─── Run ──────────────────────────────────────────────────────────────────────

.PHONY: start
start: ## Run the api and the web ui together (Ctrl-C stops both)
	@trap 'kill $$(jobs -p) 2>/dev/null' INT TERM; \
	$(MAKE) -C $(API) start & \
	$(MAKE) -C $(WEB) start & \
	wait

.PHONY: start-api
start-api: ## Run the api only
	@$(MAKE) -C $(API) start

.PHONY: start-web
start-web: ## Run the web ui only
	@$(MAKE) -C $(WEB) start

# ─── Checks ───────────────────────────────────────────────────────────────────

.PHONY: check
check: ## Run every check for both services
	@_failed=0; \
	$(MAKE) -C $(API) check || _failed=1; \
	$(MAKE) -C $(WEB) check || _failed=1; \
	exit $$_failed

.PHONY: check-api
check-api: ## Run the api checks only
	@$(MAKE) -C $(API) check

.PHONY: check-web
check-web: ## Run the web checks only
	@$(MAKE) -C $(WEB) check

# ─── Help ─────────────────────────────────────────────────────────────────────

.PHONY: help
help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
