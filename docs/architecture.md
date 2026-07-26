# Architecture and delivery boundaries

The API process owns synchronous HTTP, SSE, and WebSocket connections. Workers own durable agent workflows, scheduled sessions, document ingestion, notifications, and outbox dispatch. PostgreSQL is the system of record; Redis is cache/ephemeral coordination, SQS is delivery, S3 is document storage, and the model gateway targets a separately hosted OpenAI-compatible inference service.

## Durable flow

Every state transition and its outbox event commit in one database transaction. A worker acknowledges a queue message only after its transition commits. Handlers use the workflow ID plus operation key as their idempotency key. Bounded retries use exponential backoff; exhausted messages move to the DLQ. Operators replay only after correcting the cause and retain the original message ID for deduplication.

The public contract is `api/openapi.yaml`; stream frames are defined in `api/asyncapi.yaml`. RFC 9457-style `application/problem+json` is the only error body. Generated Go and TypeScript bindings must be derived from the same tagged contract release.

## Trust boundaries

Auth0 access tokens are verified at the API. The internal user ID is resolved from `sub`; clients never supply a tenant ID. Every user-owned transaction sets PostgreSQL tenant context and RLS remains enabled for application roles. Uploaded text is untrusted context: it is scanned, extracted in isolation, tenant-filtered, bounded, delimited, and never treated as an instruction. Objects are private and accessed using short-lived signed URLs.

Production workloads run in EKS private subnets. Public traffic terminates at an ALB using a TLS 1.3 security policy and the regional WAF. API/worker identities receive separate least-privilege AWS roles. GPU inference is internal-only and isolated on tainted nodes.

## Local development

Run `docker compose up -d` for PostgreSQL/pgvector, Redis, MinIO, Mailpit, and LocalStack SQS. The optional local model requires NVIDIA Container Toolkit and starts with `docker compose --profile model up -d`. Application binaries are intentionally run from their native toolchains for fast reloads.
