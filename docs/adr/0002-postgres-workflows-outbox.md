# ADR 0002: PostgreSQL workflows and transactional outbox

Status: accepted

PostgreSQL stores aggregate state, workflow state, invocation telemetry, and an outbox event in the same transaction. SQS messages trigger idempotent workers but do not own business truth. Redis cannot be used as durable workflow state.

This provides recoverability and auditability without adopting a second workflow platform. Queue delivery remains at-least-once, so every side effect requires a stable idempotency key and terminal transitions are immutable.
