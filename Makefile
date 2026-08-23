.PHONY: up down logs logs-api ps db redis migrate build test lint seed \
        health city-create city-get city-support badge assets-check storybook

API_URL ?= http://localhost:8080
TEST_SLUG ?= testcity

# ── Docker Compose ──────────────────────────────────────────────────────────

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f

logs-api:
	docker compose logs -f api

ps:
	docker compose ps

# ── Database ─────────────────────────────────────────────────────────────────

db:
	docker compose exec postgres psql -U bitown bitown

redis:
	docker compose exec redis valkey-cli

migrate:
	docker compose exec postgres psql -U bitown bitown \
	  -f /docker-entrypoint-initdb.d/001_init.sql

# ── Go ───────────────────────────────────────────────────────────────────────

build:
	cd api && GOOS=linux GOARCH=amd64 go build -o bitown-api ./cmd/bitown/

test:
	cd api && go test ./...

lint:
	cd api && golangci-lint run

# ── Dev / Debug ──────────────────────────────────────────────────────────────

health:
	curl -s $(API_URL)/api/health | jq .

seed: _guard-local
	curl -s -X POST $(API_URL)/api/cities \
	  -H "Content-Type: application/json" \
	  -d '{"name":"TestCity","slug":"$(TEST_SLUG)","country_code":"JP"}' | jq .

city-create: _guard-local
	@read -p "slug: " slug; \
	 read -p "name: " name; \
	 read -p "country_code [JP]: " cc; \
	 cc=$${cc:-JP}; \
	 curl -s -X POST $(API_URL)/api/cities \
	   -H "Content-Type: application/json" \
	   -d "{\"name\":\"$$name\",\"slug\":\"$$slug\",\"country_code\":\"$$cc\"}" | jq .

city-get:
	curl -s $(API_URL)/api/cities/$(TEST_SLUG) | jq .

city-support:
	curl -s -X POST $(API_URL)/api/cities/$(TEST_SLUG)/support \
	  -H "Content-Type: application/json" \
	  -d '{"sector":"pop"}' | jq .

badge:
	curl -s $(API_URL)/badge/$(TEST_SLUG).svg

assets-check:
	python3 scripts/check_assets.py

# ── Storybook (placement object catalog) ─────────────────────────────────────

storybook:
	cd web && npm install && npm run storybook

# Guard: prevents data-mutating dev commands from running against non-local URLs.
_guard-local:
	@if [ "$(API_URL)" != "http://localhost:8080" ]; then \
	  echo "ERROR: This command is for local dev only."; \
	  echo "  API_URL=$(API_URL)"; \
	  echo "  To override intentionally: make <target> _guard-local="; \
	  exit 1; \
	fi
