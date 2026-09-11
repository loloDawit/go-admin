.PHONY: help up down dev api web seed test fmt gofmtcheck lint tidy

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

up: ## Start MySQL and wait for it to accept connections
	docker compose up -d --wait

down: ## Stop MySQL (data is preserved in the dbdata volume)
	docker compose down

dev: up seed ## Start the database, seed it, then run the API
	$(MAKE) api

api: ## Run the API server
	go run .

web: ## Run the React dev server
	cd clients && npm start

seed: ## Create permissions, roles, and the owner account (idempotent)
	go run ./cmd/seed

test: ## Run all Go tests (starts its own MySQL via testcontainers)
	go test ./... -count=1

fmt: ## Format all Go source
	gofmt -w $$(find . -type f -name '*.go' -not -path './vendor/*')

gofmtcheck: ## Fail if any Go file is unformatted
	@need_fmt=$$(gofmt -l $$(find . -type f -name '*.go' -not -path './vendor/*'));\
	if [ "$$need_fmt" = "" ]; then echo "hooray"; else echo "files that need formatting:"; echo $$need_fmt; exit 1; fi

lint: gofmtcheck ## Vet and format-check
	go vet ./...

tidy: ## Tidy module dependencies
	go mod tidy
