# Deployment and rollback runbook

## Before deployment

- Confirm CI contracts, Go race tests, frontend build, Terraform validation, vulnerability scan, migration dry-run, and backup restore evidence.
- Review Terraform plan from the encrypted remote state and obtain approval for infrastructure changes.
- Publish immutable API/worker image digests and produce an environment Kustomize overlay. Populate Secrets Manager outside Terraform.

## Canary

Apply database expansion migrations first. Deploy one canary API/worker using image digests, then verify readiness, authentication, RLS isolation, queue age, workflow failures, first-token latency, SSE reconnect, and WebSocket resume. Shift traffic gradually while watching SLO burn alerts.

## Rollback

Stop traffic shift and restore the previous image digest. Pause workers if the new version emitted incompatible jobs. Do not roll back an irreversible schema migration; use expand/contract migrations and ship a forward repair. Replay failed jobs only after the older version is confirmed compatible. Record timestamps, image digests, migration versions, and affected workflow IDs.
