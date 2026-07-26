# AI Learning Coach backend

The backend is a dependency-light Go 1.22 service organized around domain,
application, ports, and adapters. The default local profile uses an in-memory
tenant-scoped repository and deterministic model, making the API runnable
without cloud credentials. `migrations/001_initial.sql` is the production
PostgreSQL/pgvector schema and enables row-level security.

## Run locally

```sh
APP_ENV=local go run ./cmd/api
curl -H 'Authorization: Bearer dev:learner::learner@example.com' \
  'http://localhost:8080/api/v1/sessions/daily?date=2026-07-26'
```

Set `LLM_BASE_URL` to route model requests to a local or hosted
OpenAI-compatible gateway. Production authentication requires
`AUTH0_ISSUER` and `AUTH0_AUDIENCE`. Development tokens are rejected unless
`APP_ENV` is explicitly `local` or `development`.

## Validate

```sh
go test ./...
go vet ./...
```

The in-memory adapter is intentionally a local/test profile. Production must
wire a PostgreSQL repository whose every tenant transaction starts with
`SET LOCAL app.user_id = $1`, plus durable outbox and object-storage adapters.
