# Incident response runbook

## Triage

1. Declare severity and incident lead; preserve trace IDs and timestamps.
2. Check availability, error rate, first-token latency, queue age/DLQ depth, database connections, Redis health, object errors, provider circuits, and GPU saturation.
3. Determine tenant impact before inspecting payloads; never paste user documents, prompts, or tokens into an incident channel.

## Containment

- Model/provider failure: open fallback circuit, reduce concurrency, and keep accepted workflows durable.
- Queue backlog: stop schedulers before scaling consumers; verify database capacity and idempotency before replay.
- Suspected tenant isolation issue: disable affected endpoints, preserve audit logs, rotate relevant credentials, and engage privacy/security response.
- Malicious upload: quarantine the object and chunks, stop ingestion for its checksum, and retain evidence according to policy.

## Recovery

Restore service from a known-good image/configuration, drain DLQs in a controlled batch, and verify a synthetic authenticated learning flow. Close only when error budgets stabilize and no duplicate lessons, XP, revisions, or notifications were created. Produce a blameless review with corrective owners and dates.
