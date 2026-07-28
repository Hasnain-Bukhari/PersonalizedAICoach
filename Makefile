.PHONY: dev build test lint frontend backend infra-up infra-down

dev:
	pnpm dev

build:
	pnpm build

test:
	pnpm test

lint:
	pnpm lint

frontend:
	pnpm --filter frontend dev

backend:
	cd backend && go run ./cmd/api

infra-up:
	docker compose up -d postgres redis minio mailpit elasticmq

infra-down:
	docker compose down
