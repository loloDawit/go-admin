# Each checkout gets its own compose project. Without this, a git worktree
# running the stack drives the SAME containers and volume as the main checkout:
# a worktree on an older branch once ran its migrate job against a database the
# main checkout had already migrated forward, and both sessions lost containers
# under each other. Host ports still collide, so a second stack now fails loudly
# on the port bind rather than silently adopting the first one's containers.
COMPOSE_PROJECT_NAME := $(notdir $(CURDIR))-$(shell printf '%s' "$(CURDIR)" | shasum | cut -c1-6)
export COMPOSE_PROJECT_NAME

COMPOSE := docker compose -f deploy/compose/docker-compose.yml
GO_PKGS := ./...

# One definition for the stack and for the tests that assert against it.
# Compose reads these via ${VAR:-default}; the integration suite reads them
# from the environment and fails if absent rather than defaulting, so the two
# cannot drift apart silently.
export OWNER_EMAIL       ?= owner@example.com
export OWNER_PASSWORD    ?= dev_only_owner_password
export SESSION_CACHE_TTL ?= 10s

.PHONY: help hooks generate up down down-v dev logs ps psql seed test test-unit test-integration fmt lint tidy

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

up: ## Boot the stack and wait for containers to report ready
	$(COMPOSE) up -d --build --wait

down-v: ## Stop the stack and delete its volumes
	$(COMPOSE) down -v --remove-orphans

down: ## Stop the stack (volumes preserved)
	$(COMPOSE) down

generate: ## Regenerate code from published contracts
	go run ./tools/permgen
	go run ./tools/permgen -out services/orders/internal/permission/permission_gen.go -package permission
	cd apps/web && npm ci && npm run generate

seed: up ## Create the first owner account (idempotent)
	$(COMPOSE) --profile seed run --rm identity-seed

ps: ## Show this checkout's containers
	$(COMPOSE) ps

psql: ## Open psql against a service's database, e.g. make psql DB=catalog
	$(COMPOSE) exec postgres psql "postgres://$(DB)_user:dev_only_$(DB)@127.0.0.1/$(DB)_db"

dev: up ## Boot the stack and show gateway logs
	$(COMPOSE) logs -f gateway

logs: ## Tail all service logs
	$(COMPOSE) logs -f

test: test-unit test-integration test-e2e ## Run everything

test-unit: ## Unit tests, no stack required
	go test $(GO_PKGS) ./test/arch/... -count=1

test-integration: seed ## Integration tests against the running stack
	# test/integration carries //go:build integration; drop -tags integration
	# and go test matches no packages here at all (a hard failure, not a
	# silent pass) — but the same mistake against a wider path like ./test/...
	# would quietly run 0 integration tests and report ok, which is why the
	# tag stays pinned to this exact target rather than something broader.
	go test -tags integration ./test/integration/... -count=1

test-e2e: seed ## Browser tests against the app the gateway serves
	# BASE_URL points Playwright at the gateway's own build. Against the Vite
	# dev server this proves only that the source compiles, not that what
	# ships works.
	cd apps/web && npm ci && npx playwright install --with-deps chromium && BASE_URL=http://localhost:8080 npx playwright test

load: ## Run the k6 load test against the running stack
	# Every VU is the same member of staff and the limiter is per principal,
	# so this run raises it: the target being measured is the order path.
	RATE_LIMIT_PER_SECOND=100000 RATE_LIMIT_BURST=100000 $(MAKE) up
	docker run --rm --network host -v $(PWD)/deploy/k6:/scripts \
		-e BASE_URL=http://localhost:8080 \
		-e OWNER_EMAIL=$(OWNER_EMAIL) -e OWNER_PASSWORD=$(OWNER_PASSWORD) \
		grafana/k6:0.54.0 run /scripts/order-path.js

fmt: ## Format
	gofmt -w services platform test

lint: ## Vet and format check
	@unformatted=$$(gofmt -l services platform test); \
	if [ -n "$$unformatted" ]; then echo "needs gofmt:"; echo "$$unformatted"; exit 1; fi
	go vet $(GO_PKGS) ./test/arch/...
	go vet -tags integration ./test/integration/...

tidy: ## Tidy modules
	go mod tidy

hooks: ## Install the repository git hooks
	@git config core.hooksPath .githooks
	@echo "core.hooksPath -> .githooks"
