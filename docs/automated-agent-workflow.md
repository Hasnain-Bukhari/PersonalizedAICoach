# Automated agent workflow

Start Codex from this repository root and ask it to use `verified-delivery`. Feature work pauses after acceptance criteria and one implementation plan; application edits begin only after your explicit approval. Once approved, Codex continues through testing, repair, browser QA, and independent review.

To verify manual work without Codex, run `./scripts/verify-changes.sh`. To run the same complete workflow directly, run `./scripts/verify.sh`. Open the browser report with `pnpm test:e2e:report`. Logs, screenshots, traces, and the stable summary are under `artifacts/`; the latest summary is `artifacts/reports/latest-verification.md`. The default E2E mode uses the real local API. `E2E_DEMO_MODE=true pnpm test:e2e` is only a UI-harness diagnostic and does not replace full verification.

`PASS` means every applicable required check ran and passed. `PARTIAL` means delivered work exists but an applicable check was blocked. `FAIL` means required behavior or a required check failed. Credentials and protected external dependencies must be reported as blockers; never substitute production resources or embed secrets.

## Future feature prompt

```text
Use the `verified-delivery` workflow.

Task:
[DESCRIBE THE FEATURE OR BUG]

First:
- inspect the repository
- capture and preserve my existing manual changes
- define explicit acceptance criteria
- use focused read-only exploration
- provide one implementation plan
- wait for my approval before modifying application code

After I approve:
- implement the plan
- add or update tests
- run focused checks
- run the complete repository verification workflow
- start the real local application
- verify affected user flows with Playwright
- inspect console errors, failed requests, and service logs
- repair failures
- perform an independent review
- rerun full verification
- return PASS, PARTIAL, or FAIL with evidence

Never discard or silently overwrite my manual changes.
```

## Verify my manual changes prompt

```text
Use the `verified-delivery` workflow in verify-only mode.

I manually changed files in this repository.
Do not reset, revert, discard, or overwrite my work.

Capture the current Git status, inspect every modified and untracked file, derive affected behavior, run the complete frontend/backend/API/database/Playwright workflow, inspect console errors, failed requests, and logs, and return PASS, PARTIAL, or FAIL with evidence. Do not repair code unless I explicitly ask you to.
```

## Verify and fix my manual changes prompt

```text
Use the `verified-delivery` workflow in verify-only-and-repair mode.

I manually changed files in this repository. Do not reset, revert, discard, or overwrite my intended work. Capture the Git state, inspect all changes, derive acceptance criteria, run complete verification, diagnose failures, apply the smallest valid fixes, update tests when necessary, verify affected browser flows, inspect console/network/service evidence, repeat until passed or genuinely blocked, perform independent review, and return PASS, PARTIAL, or FAIL.
```
