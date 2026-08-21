GOTESTSUM_VERSION := v1.13.0
GOTESTSUM := $(shell go env GOPATH)/bin/gotestsum
AIR := $(shell go env GOPATH)/bin/air
ENV_FILE := $(abspath ../../.env)
MAX_FILE_LINES := 250

SHELL := /bin/bash
.DEFAULT_GOAL := help

# ─── Setup & Install ──────────────────────────────────────────────────────────

.PHONY: setup
setup: ## Install the toolchain (go, air, golangci-lint, gotestsum)
	@if ! command -v go >/dev/null 2>&1; then brew install go; else echo "go: $$(go version)"; fi
	@if ! command -v golangci-lint >/dev/null 2>&1; then brew install golangci-lint; else echo "golangci-lint: $$(golangci-lint --version)"; fi
	@if ! command -v air >/dev/null 2>&1; then go install github.com/air-verse/air@latest; else echo "air: installed"; fi
	@if [ ! -x "$(GOTESTSUM)" ]; then go install gotest.tools/gotestsum@$(GOTESTSUM_VERSION); else echo "gotestsum: $$($(GOTESTSUM) --version)"; fi

.PHONY: install
install: ## Download and verify module dependencies
	@go mod download
	@go mod verify

# ─── Run ──────────────────────────────────────────────────────────────────────

.PHONY: start
start: ## Run the API with hot reload
	@$(AIR) -c .air.toml

.PHONY: dev
dev: start ## Alias for start

.PHONY: run
run: ## Run the API once (no reload)
	@set -a && [ -f $(ENV_FILE) ] && . $(ENV_FILE); set +a; go run ./cmd/server

# ─── Checks ───────────────────────────────────────────────────────────────────

.PHONY: check
check: ## Run every check in parallel (vet + lint + file-length + unit + feature + component)
	@set -o pipefail; \
	_dir=$$(mktemp -d); trap 'rm -rf $$_dir' EXIT; \
	echo "▸ launching 6 checks in parallel (output captured per step, dumped on completion)"; \
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
	step vet  "go vet"                                            go vet ./... & \
	step lint "golangci-lint"                                     golangci-lint run ./... & \
	step flen "file length ($(MAX_FILE_LINES) line hard limit)"   $(MAKE) -s check-file-length & \
	step unit "unit"                                              $(GOTESTSUM) --format testdox -- ./internal/... -count=1 & \
	step feat "feature"                                           $(GOTESTSUM) --format testdox -- ./tests/feature/... -count=1 -parallel=20 & \
	( set -a && [ -f $(ENV_FILE) ] && . $(ENV_FILE); set +a; \
	  step comp "component"                                       $(GOTESTSUM) --format testdox -- ./tests/component/... -count=1 ) & \
	wait; \
	echo ""; \
	for id in vet lint flen unit feat comp; do \
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
	for id in vet lint flen unit feat comp; do \
		rc=$$(cat "$$_dir/$$id.rc"); dur=$$(cat "$$_dir/$$id.dur"); label=$$(cat "$$_dir/$$id.label"); \
		done_line=$$(grep '^DONE' "$$_dir/$$id.out" 2>/dev/null || true); \
		mark="✓"; if [ "$$rc" != "0" ]; then mark="✗"; _failed=1; fi; \
		printf "  %s %s (%ss)" "$$mark" "$$label" "$$dur"; \
		if [ -n "$$done_line" ]; then printf ": %s" "$$done_line"; fi; \
		echo ""; \
	done; \
	exit $$_failed

.PHONY: check-ci
check-ci: ## Run checks for CI/Docker (no component tests — no infra available)
	@set -o pipefail; \
	_summary=$$(mktemp); _failed=0; trap 'rm -f $$_summary' EXIT; \
	run_step() { \
		local label="$$1"; shift; \
		echo "--- $$label ---"; \
		if "$$@"; then echo "  ✓ $$label" >> $$_summary; else echo "  ✗ $$label" >> $$_summary; _failed=1; fi; \
	}; \
	run_test_step() { \
		local label="$$1"; shift; \
		echo "--- $$label ---"; \
		local tmp; tmp=$$(mktemp); \
		"$$@" 2>&1 | tee "$$tmp"; local rc=$${PIPESTATUS[0]}; \
		local done_line; done_line=$$(grep '^DONE' "$$tmp" || true); \
		rm -f "$$tmp"; \
		if [ "$$rc" -eq 0 ]; then echo "  ✓ $$label: $$done_line" >> $$_summary; \
		else echo "  ✗ $$label: $$done_line" >> $$_summary; _failed=1; fi; \
	}; \
	run_step "go vet" go vet ./...; \
	run_step "golangci-lint" golangci-lint run ./...; \
	run_step "file length ($(MAX_FILE_LINES) line hard limit)" $(MAKE) check-file-length; \
	run_test_step "unit" $(GOTESTSUM) --format testdox -- ./internal/... -count=1; \
	run_test_step "feature" $(GOTESTSUM) --format testdox -- ./tests/feature/... -count=1 -parallel=20; \
	echo ""; \
	echo "═══════════════════════════════════════"; \
	echo "  make check-ci — summary"; \
	echo "═══════════════════════════════════════"; \
	cat $$_summary; \
	exit $$_failed

.PHONY: check-file-length
check-file-length: ## Fail if any Go source file exceeds $(MAX_FILE_LINES) lines (test files excluded)
	@violations=0; \
	for f in $$(find . -name '*.go' -not -name '*_test.go' -not -path '*/vendor/*'); do \
		lines=$$(wc -l < "$$f"); \
		if [ "$$lines" -gt $(MAX_FILE_LINES) ]; then \
			echo "FAIL: $$f has $$lines lines (limit: $(MAX_FILE_LINES))"; \
			violations=$$((violations + 1)); \
		fi; \
	done; \
	if [ "$$violations" -gt 0 ]; then \
		echo "$$violations file(s) exceed the $(MAX_FILE_LINES)-line limit."; exit 1; \
	fi; \
	echo "All files within $(MAX_FILE_LINES)-line limit."

# ─── Tests ────────────────────────────────────────────────────────────────────

.PHONY: test
test: ## Component tests — need the infra from .env
	@set -a && [ -f $(ENV_FILE) ] && . $(ENV_FILE); set +a; $(GOTESTSUM) --format testdox -- ./tests/component/... -count=1 $(if $(FILTER),-run $(FILTER),)

.PHONY: test-unit
test-unit: ## Unit tests — package-local, mocked deps
	@$(GOTESTSUM) --format testdox -- ./internal/... -count=1 $(if $(FILTER),-run $(FILTER),)

.PHONY: test-feature
test-feature: ## Feature tests — full scenarios against in-memory fakes
	@$(GOTESTSUM) --format testdox -- ./tests/feature/... -count=1 -parallel=20 $(if $(FILTER),-run $(FILTER),)

# ─── Lint ─────────────────────────────────────────────────────────────────────

.PHONY: lint
lint: ## Run golangci-lint
	@golangci-lint run ./...

# ─── Help ─────────────────────────────────────────────────────────────────────

.PHONY: help
help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
