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
check: ## Run every check for both services in parallel (fails if either fails)
	@set -o pipefail; \
	_dir=$$(mktemp -d); trap 'rm -rf $$_dir' EXIT; \
	echo "▸ checking 2 services in parallel (output captured per service, dumped on completion)"; \
	echo ""; \
	step() { \
		local id="$$1" label="$$2"; shift 2; \
		local start; start=$$(date +%s); \
		echo "[start] $$label"; \
		( "$$@" ) > "$$_dir/$$id.out" 2>&1; \
		local rc=$$?; \
		echo "$$rc"    > "$$_dir/$$id.rc"; \
		echo "$$label" > "$$_dir/$$id.label"; \
		echo "$$(($$(date +%s) - start))" > "$$_dir/$$id.dur"; \
		if [ "$$rc" -eq 0 ]; then echo "[done]  $$label ($$(cat $$_dir/$$id.dur)s)"; \
		else echo "[FAIL]  $$label ($$(cat $$_dir/$$id.dur)s, rc=$$rc)"; fi; \
	}; \
	step api "api  (make check)"     $(MAKE) -C $(API) check & \
	step web "web  (npm run check)"  $(MAKE) -C $(WEB) check & \
	wait; \
	echo ""; \
	for id in api web; do \
		echo "════════════════════════════════════════"; \
		echo "  $$(cat $$_dir/$$id.label)"; \
		echo "════════════════════════════════════════"; \
		cat "$$_dir/$$id.out"; \
		echo ""; \
	done; \
	_failed=0; \
	echo "═══════════════════════════════════════"; \
	echo "  make check — summary"; \
	echo "═══════════════════════════════════════"; \
	for id in api web; do \
		rc=$$(cat "$$_dir/$$id.rc"); dur=$$(cat "$$_dir/$$id.dur"); label=$$(cat "$$_dir/$$id.label"); \
		mark="✓"; if [ "$$rc" != "0" ]; then mark="✗"; _failed=1; fi; \
		printf "  %s %s (%ss)\n" "$$mark" "$$label" "$$dur"; \
	done; \
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
