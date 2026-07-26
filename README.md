# AI Learning Coach

AI Learning Coach is a Go and Vue platform for adaptive technical learning. It builds daily sessions from learner mastery, teaches through structured lessons, evaluates mixed quizzes, schedules spaced review, indexes private study material, and conducts stateful system-design interviews.

## Implementation status

This repository was rebuilt from a disconnected TypeScript prototype into a runnable Go/Vue learning platform. The inherited code, completed work, verification evidence, feature-by-feature status, and remaining production gates are documented in the [implementation report](docs/implementation-report.md).

| Area | Status |
|---|---|
| Go API, learning policies, quizzes, interviews, and local workflows | Runnable |
| Vue learner application and real API integration | Runnable |
| PostgreSQL/pgvector schema and tenant RLS policies | Implemented and migration-tested |
| OpenAPI, AsyncAPI, Compose, Kubernetes, Terraform, and CI foundations | Implemented |
| Production PostgreSQL/outbox/SQS/S3 runtime adapters | Remaining production gate |
| Binary document extraction, scanning, embeddings, and notification delivery | Remaining production gate |

## Architecture

- `backend/`: Go API and worker, clean domain/application/adapter boundaries, workflow state machines, Auth0 JWT validation, model gateway, learning policies, RAG, interviews, and PostgreSQL migrations.
- `frontend/`: Vue 3, Pinia, and Vite learner experience for today’s itinerary, lessons, quizzes, progress, documents, interviews, preferences, and notifications.
- `api/`: OpenAPI 3.1 REST contract and AsyncAPI streaming contract.
- `infrastructure/`: local Compose services, AWS Terraform foundation, and Kubernetes/Kustomize workloads.
- `docs/`: architectural decisions, deployment guidance, and operational runbooks.

The runnable development profile uses a tenant-scoped in-memory adapter so contributors can exercise the complete API without cloud credentials. The PostgreSQL/pgvector schema and RLS policies define the production persistence contract; a durable PostgreSQL/queue adapter remains a production integration gate and should not be confused with the in-memory profile.

## Prerequisites

- Node.js 18 or newer and pnpm 8 or newer
- Go 1.22 or newer
- Docker for local infrastructure

## Quick start

```bash
pnpm install
make infra-up
make dev
```

The frontend runs on `http://localhost:5173` and proxies `/api` to the Go API on `http://localhost:8080`.

For local authentication, explicitly configure the API for development and send `Authorization: Bearer dev:<subject>`. Production mode fails closed unless Auth0 issuer and audience settings are present.

```bash
cd backend
APP_ENV=development go run ./cmd/api
```

Enable frontend demonstration data only when intentionally working without an API:

```bash
VITE_DEMO_MODE=true pnpm --filter frontend dev
```

## Quality gates

```bash
make build
make test
make lint
docker compose config --quiet
kubectl kustomize infrastructure/k8s/base >/dev/null
```

Backend-only verification:

```bash
cd backend
go test ./...
go vet ./...
```

## Configuration

Copy `.env.example` and set only non-secret local values. Production secrets belong in AWS Secrets Manager or a Kubernetes Secret populated by the deployment system. Important settings include:

- `APP_ENV`, `AUTH0_ISSUER`, and `AUTH0_AUDIENCE`
- `LLM_BASE_URL` and task-specific `LLM_*_MODEL` aliases
- `DATABASE_URL`, `REDIS_URL`, queue names, object bucket, and AWS region
- `VITE_API_URL` and the opt-in `VITE_DEMO_MODE`

See the [implementation report](docs/implementation-report.md), [architecture](docs/architecture.md), [deployment runbook](docs/runbooks/deployment.md), and contracts in [`api/`](api/) for details.

## Current delivery boundary

The repository now supplies the product-shaped learning loop, contracts, UI, policies, security boundaries, and infrastructure foundation. Before a production launch, complete and load-test the PostgreSQL/outbox/SQS/object-storage adapters, isolated binary document extraction and malware scanning, provider failover, notification delivery, and deployment-specific IAM/ingress wiring described in the [implementation report](docs/implementation-report.md#remaining-production-gates) and runbooks.
