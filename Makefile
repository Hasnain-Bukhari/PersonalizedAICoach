.PHONY: dev build test lint frontend backend infra-up infra-down

dev:
	npm run dev

build:
	npm run build

test:
	npm test

lint:
	npm run lint

frontend:
	npm --prefix frontend run dev

backend:
	cd backend && go run ./cmd/api

infra-up:
	docker compose up -d postgres redis minio mailpit elasticmq

infra-down:
	docker compose down
