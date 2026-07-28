# Repository delivery workflow

Apply these rules to every feature, bug fix, refactor, and verification request.

## Protect the working tree

- Begin with `git status --short`. Record modified and untracked paths as pre-existing user work.
- Never reset, restore, discard, overwrite, or silently revert manual work. Do not delete untracked files.
- Distinguish pre-existing files from files changed during the current task and report both groups.
- Verify the complete working tree when manual changes can affect the application.

## Plan before application edits

1. Understand the request and write explicit, testable acceptance criteria.
2. For complex work, assign focused read-only exploration to the `explorer` agent.
3. Present one implementation plan and wait for explicit user approval before editing application code.
4. After approval, continue autonomously through implementation, tests, repair, QA, review, and reporting.

Repository-local automation/configuration maintenance may proceed without this approval pause when the user already requested it explicitly.

## Agent boundaries

- `explorer`: read-only discovery, data-flow tracing, affected tests, and risks.
- `implementer`: bounded approved edits and focused tests. Never overlap another editor's files.
- `qa`: independent execution of builds, tests, services, browser checks, console/network inspection, and logs.
- `failure-investigator`: one failure or log set, root cause, and smallest repair recommendation.
- `reviewer`: read-only final working-tree review for correctness, security, regressions, maintainability, and test quality.

Keep the parent thread to objectives, criteria, plans, decisions, summaries, verification state, and final evidence. Put long output under `artifacts/`. Agent replies must give a conclusion, evidence, paths, and next action—not raw logs.

## Verification and repair

Run every applicable gate: dependency validation, format check, lint, typecheck, frontend/backend build, unit/integration tests, migration validation, service startup, readiness/health checks, Playwright scenarios, browser console and failed-request inspection, service-log inspection, and independent review.

- Use `NOT_APPLICABLE` only with a concrete reason.
- Use `BLOCKED` or `SKIPPED` when an applicable check cannot run; never count it as passed.
- On failure, record command and exit code, preserve output, isolate the cause, apply the smallest approved fix, rerun the focused check, then rerun full verification. Limit complete repair cycles to five unless the user asks otherwise.
- Never weaken valid tests, hide browser failures, mock away required behavior, or claim success from inspection alone.

Use exactly one final status: `PASS` when every applicable check ran and passed; `PARTIAL` when implementation exists but a required check is blocked; `FAIL` when required behavior or verification fails.

Use the repository `verified-delivery` skill for the complete feature and verify-only procedures. The primary command is `./scripts/verify.sh`; `./scripts/verify-changes.sh` makes manual-change intent explicit.
