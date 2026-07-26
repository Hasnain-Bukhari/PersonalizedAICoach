# AI Learning Coach implementation report

**Report date:** 26 July 2026  
**Purpose:** Record the inherited repository state, the work completed during the Go/Vue rebuild, the verification performed, and the remaining production gates.

## Executive summary

The repository began as a partially generated TypeScript monorepo containing several disconnected experiments. Its active backend returned only `Hello World`, its active Vue route displayed a static welcome page, and its database, service, and frontend prototypes did not form a working product.

The repository has now been reorganized around a Go backend, a Vue 3 learner application, versioned API contracts, PostgreSQL/pgvector schema design, local service dependencies, and AWS/Kubernetes deployment foundations. A complete local vertical slice is runnable: authenticate as a development learner, generate a daily session, render the lesson, submit the quiz, update mastery and spaced repetition, record XP/activity, manage text study documents, and conduct a stateful system-design interview.

The current runtime deliberately uses tenant-scoped in-memory persistence. The production data schema and infrastructure exist, but the PostgreSQL/outbox/SQS/S3 adapters and several operational integrations remain production gates.

## What was already in the repository

### Repository structure

The inherited project was a TypeScript workspace with three packages:

- An Express backend under `backend/`.
- A Vue 3/Vite frontend under `frontend/`.
- A `shared/` package containing basic user, goal, coaching-session, plan, and task types.

The high-level choice of Vue, Express, shared DTOs, and PostgreSQL prototypes showed the intended direction, but the packages were not consistently integrated.

### Backend state

The active backend entry point:

- Started Express on port 3001.
- Parsed JSON requests.
- Exposed only a root route returning `Hello World`.

Other backend files were disconnected prototypes rather than active capabilities:

- A memory-only scheduler with `Review`, `Practice`, and `Quiz` tasks.
- Mock quiz evaluation based on an `isCorrect` property supplied in input.
- Notification and rescheduling functions that only wrote console messages.
- An OpenAI service prototype in a malformed `llmService.ts**` filename.
- Plan and task route prototypes referencing Prisma models and `req.user` fields that did not exist.
- Simultaneous Prisma, TypeORM, and Mongoose experiments.
- An incomplete Prisma schema whose models did not match its routes or seed script.

### Frontend state

The active Vue router rendered only a static home page. Additional components existed, but were not mounted into the application:

- A daily workspace with hard-coded video, lesson, and quiz content.
- A chat component opening an SSE endpoint that the backend did not provide.
- A manual learning-plan form that only logged values.
- A progress dashboard with hard-coded topics, streaks, and calendar events.
- A notification component placed incorrectly inside the backend source tree.
- A React `App.tsx**` prototype inside the Vue application.

Several frontend prototypes imported packages that were not declared in the active package manifest.

### Tests, build, and infrastructure

The inherited repository did not have a reliable quality gate:

- `npm run build` failed on TypeScript syntax and type errors.
- Backend test files imported Vitest and AJV without a configured test script or required dependencies.
- The SSE test attempted to parse a full SSE frame directly as JSON.
- Generated JavaScript, declarations, and source maps were committed beside TypeScript source files.
- Files with literal `**` suffixes duplicated manifests, routes, entry points, migrations, and UI files.
- npm and pnpm lock/install strategies were mixed.
- Docker Compose described only minimal frontend/backend containers and did not provide the required data, queue, storage, email, or model services.
- No AWS Terraform, Kubernetes workload, OpenAPI contract, AsyncAPI contract, CI workflow, security workflow, or operational runbook existed.

## What has been completed

### 1. Repository recovery

- Removed the obsolete TypeScript backend, contradictory ORM prototypes, generated compiler output, literal `**` files, disconnected UI prototypes, old verification scripts, and obsolete `shared/` package.
- Standardized the frontend workspace and added a pnpm lockfile.
- Added root `Makefile` and npm commands for development, build, test, lint, and local infrastructure.
- Replaced the historical setup notes with current architecture and operating instructions.

### 2. Go backend replacement

The backend is now a Go 1.22 module organized into explicit boundaries:

```text
backend/
├── cmd/api                 HTTP, SSE, and WebSocket process
├── cmd/worker              Worker deployment boundary
├── internal/domain         Learning entities and deterministic policies
├── internal/application    Daily, quiz, RAG, and interview use cases
├── internal/ports          Persistence, model, object, and clock interfaces
├── internal/adapters       Identity, model, and in-memory implementations
├── internal/api            Versioned transport handlers and middleware
└── migrations              PostgreSQL/pgvector production schema
```

Completed backend behavior includes:

- Auth0 RS256/JWKS verification with issuer, audience, expiry, and subject checks.
- Development tokens only when `APP_ENV` explicitly selects a local or development environment.
- Production startup failure when Auth0 settings are missing.
- UUID identifiers and snake_case JSON contracts.
- REST endpoints for daily sessions, session completion, quizzes, documents, interviews, analytics, and preferences.
- Authenticated SSE workflow events with a per-user monotonic resume cursor.
- Browser-compatible WebSocket authentication using an encoded subprotocol rather than query-string tokens.
- Explicit daily and interview workflow states.
- Provider-neutral OpenAI-compatible model gateway with task-specific model aliases.
- Model-generated structured lesson/quiz content with validation and deterministic fallback content.
- Tenant-filtered in-memory repositories for the runnable development profile.
- Daily planning that prioritizes due or weak knowledge nodes and honors configured session duration.
- Mixed quiz evaluation, explanations, misconception feedback, mastery updates, confidence tracking, and scoped idempotency.
- The specified SM-2 variant, including ease-factor updates and review intervals.
- Idempotent XP, timezone-aware streaks, and study-activity aggregation.
- Text/Markdown upload validation, SHA-256 checksums, 800-word chunks with 120-word overlap, retrieval, and citations.
- Explicit `requires_ocr` status for binary formats that need the isolated extractor.
- Stateful multi-stage system-design interviews and rubric scorecards.
- RFC-style `application/problem+json` error responses.

### 3. Database and tenant security design

The initial PostgreSQL migration now defines the full production-shaped data model:

- Users, preferences, goals, and notification preferences.
- Domains, topics, prerequisites, and learner knowledge nodes.
- Workflows, daily sessions, steps, lessons, and lesson sources.
- Quizzes, questions, attempts, responses, revisions, and revision history.
- Interviews, messages, and scorecards.
- Documents, pgvector chunks, and long-term memory.
- Study activity, XP ledger, badges, and streak snapshots.
- Agent runs, transactional outbox events, and notifications.

The migration enables and forces row-level security for direct user-owned tables and adds join-based policies for tenant-owned child tables. The rollback migration removes only objects owned by this application rather than dropping the entire public schema.

The migration was executed successfully against PostgreSQL 16 with pgvector during verification.

### 4. Vue learner application

The Vue 3 application now provides routed, responsive experiences for:

- Today’s adaptive itinerary.
- Structured lesson content, Mermaid architecture text, pitfalls, cheat sheets, and citations.
- Mixed quiz questions with backend submission and evaluation feedback.
- Knowledge mastery, revision queue, readiness, XP, streak, and activity views.
- Private document upload, processing status, refresh, and deletion.
- Multi-stage live system-design interviews and scorecards.
- Coaching mode, domains, timezone, duration, schedule, and notification preferences.
- In-app notifications and visible loading, authentication, connection, and validation failures.

The real API is the default. Demonstration fixtures are used only when `VITE_DEMO_MODE=true` is explicitly configured. The Vite server proxies both HTTP and WebSocket traffic to the Go API.

### 5. Contracts and infrastructure

Added delivery artifacts include:

- OpenAPI 3.1 REST contract aligned with the Go and Vue payloads.
- AsyncAPI contract for workflow SSE events and interview WebSocket frames.
- Local Compose services for PostgreSQL/pgvector, Redis, MinIO, Mailpit, SQS/DLQ emulation, and optional vLLM inference.
- AWS Terraform foundation for networking, EKS application/GPU nodes, Aurora, Redis, S3/KMS, SQS/DLQ, Secrets Manager, SES, and WAF.
- Kubernetes/Kustomize workloads for API, worker, and GPU inference, including probes, autoscaling, disruption budgets, network policies, service accounts, configuration, and secret templates.
- CI workflows for Go, Vue, contracts, Terraform, Kubernetes, and Compose validation.
- Security scanning workflow using Trivy and SARIF upload.
- Architecture decision records plus deployment, incident-response, and backup/restore runbooks.

## Feature coverage

| Capability | Current status | Notes |
|---|---|---|
| Daily session | Runnable | Synchronous in local memory profile; durable worker execution remains pending. |
| Adaptive lesson | Runnable | Uses the model gateway with schema validation and deterministic fallback. |
| Quiz and explanations | Runnable | Backend evaluation, mastery, SM-2, XP, and idempotency are implemented. |
| Knowledge graph | Runnable foundation | Learner nodes update from quiz results; curriculum publication tooling remains pending. |
| System-design interview | Runnable | Multi-stage WebSocket conversation and scorecard are implemented. |
| Text/Markdown RAG | Runnable foundation | Tenant filtering, chunking, retrieval, and citations are implemented. |
| PDF/DOCX/PPTX ingestion | Production integration required | Upload/status exists; isolated extraction, scanning, OCR, embeddings, and reranking remain. |
| Analytics | Runnable foundation | Mastery, readiness, XP, streak, due reviews, and activity are available. |
| Badges and levels | Schema/UI foundation | Award criteria and durable badge processing remain. |
| In-app notifications | UI/schema foundation | Durable scheduling and dispatch remain. |
| Email notifications | Infrastructure foundation | SES/Mailpit foundations exist; delivery adapter and templates remain. |
| Auth0 and tenant boundaries | Implemented at API/schema level | Production persistence adapter must set RLS tenant context transactionally. |
| PostgreSQL persistence | Schema complete | Runtime currently uses the in-memory adapter. |
| Queue/outbox processing | Schema/infrastructure foundation | Worker consumer, retry, and DLQ replay implementation remain. |
| AWS deployment | Foundation | Environment-specific IAM, ingress, DNS, certificates, image digests, and canary automation remain. |

## Verification evidence

The following checks passed after implementation:

- Go formatting with `gofmt`.
- `go test ./...` across domain, application, memory adapter, and API packages.
- `go vet ./...`.
- Vue TypeScript validation and Vite production build.
- Frontend test and lint scripts.
- Authenticated API smoke test for health and daily-session generation.
- PostgreSQL 16/pgvector initialization using the production migration.
- YAML parsing for OpenAPI, AsyncAPI, and Compose files.
- `docker compose config --quiet`.
- `kubectl kustomize infrastructure/k8s/base`.
- `git diff --check`.

Terraform was not available in the local host environment; CI is configured to run `terraform init`, `terraform validate`, and the contract linters.

## Remaining production gates

The next implementation phase should address these items in order:

1. Implement the PostgreSQL adapter and set `SET LOCAL app.user_id` inside every tenant transaction.
2. Commit workflow transitions and outbox messages atomically, then implement SQS consumers, retries, idempotent acknowledgements, and DLQ replay.
3. Connect S3/MinIO object storage, malware scanning, isolated PDF/DOCX/PPTX extraction, embeddings, hybrid search, and reranking.
4. Persist agent-run telemetry, prompt versions, citations, token usage, retry counts, and latency.
5. Implement scheduled daily generation and in-app/email notification delivery.
6. Add provider fallback adapters, circuit breakers, concurrency controls, and cost budgets.
7. Complete deployment-specific IAM, ingress, TLS, DNS, WAF attachment, alarms, observability, canary rollout, and restore testing.
8. Add browser E2E, tenant-isolation integration, contract, load, failure-injection, and security test suites.

## Source map

- Product entry point: [`README.md`](../README.md)
- Go backend guide: [`backend/README.md`](../backend/README.md)
- REST contract: [`api/openapi.yaml`](../api/openapi.yaml)
- Streaming contract: [`api/asyncapi.yaml`](../api/asyncapi.yaml)
- Database schema: [`backend/migrations/001_initial.sql`](../backend/migrations/001_initial.sql)
- Architecture: [`docs/architecture.md`](architecture.md)
- Deployment runbook: [`docs/runbooks/deployment.md`](runbooks/deployment.md)
- Local dependencies: [`docker-compose.yml`](../docker-compose.yml)
- AWS foundation: [`infrastructure/terraform`](../infrastructure/terraform)
- Kubernetes foundation: [`infrastructure/k8s`](../infrastructure/k8s)
