COMPOSE := docker compose -f deploy/compose/docker-compose.yml
# Excludes the repo root on purpose: the legacy packages still live there and
# their tests need MySQL. Widening this to ./... breaks test/lint until M4
# deletes the legacy tree.
GO_PKGS := ./services/... ./platform/...

.PHONY: help up down dev logs test test-unit test-integration fmt lint tidy

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

up: ## Boot the stack and wait for containers to report ready
	$(COMPOSE) up -d --build --wait

down: ## Stop the stack (volumes preserved)
	$(COMPOSE) down

dev: up ## Boot the stack and show gateway logs
	$(COMPOSE) logs -f gateway

logs: ## Tail all service logs
	$(COMPOSE) logs -f

test: test-unit test-integration ## Run everything

test-unit: ## Unit tests, no stack required
	go test $(GO_PKGS) ./test/arch/... -count=1

test-integration: up ## Integration tests against the running stack
	# test/integration carries //go:build integration; drop -tags integration
	# and go test matches no packages here at all (a hard failure, not a
	# silent pass) — but the same mistake against a wider path like ./test/...
	# would quietly run 0 integration tests and report ok, which is why the
	# tag stays pinned to this exact target rather than something broader.
	go test -tags integration ./test/integration/... -count=1

fmt: ## Format
	gofmt -w services platform test

lint: ## Vet and format check
	@unformatted=$$(gofmt -l services platform test); \
	if [ -n "$$unformatted" ]; then echo "needs gofmt:"; echo "$$unformatted"; exit 1; fi
	go vet $(GO_PKGS) ./test/arch/...
	go vet -tags integration ./test/integration/...

tidy: ## Tidy modules
	go mod tidy
