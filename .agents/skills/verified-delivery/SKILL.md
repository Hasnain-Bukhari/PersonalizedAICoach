---
name: verified-delivery
description: Deliver and verify repository changes with protected Git baselines, approval-gated planning, specialist agents, complete automated testing, real local services, Playwright browser inspection, repair loops, and evidence-based review. Use for feature development, bug fixes, refactors, full-stack changes, manual-change verification, or verify-and-fix requests that require comprehensive frontend, backend, API, database, and browser validation.
---

# Verified delivery

Keep detailed logs in `artifacts/`; keep the main thread concise. Follow root `AGENTS.md` and preserve all pre-existing work.

## Select a mode

- **Feature mode:** inspect, define acceptance criteria, plan, wait for approval, implement, test, repair, review, report.
- **Verify-only mode:** treat every current change as user-authored, inspect affected behavior, run verification, and do not edit unless the user explicitly asks to repair.
- **Verify-only-and-repair mode:** preserve intended manual behavior and apply only the smallest fixes needed after evidence identifies a failure.

## Run the phases

1. **Baseline:** record repository root, branch, commit, `git status --short`, tracked diff, and untracked files. Separate pre-existing work from current-task changes.
2. **Understand:** translate the request and changed files into explicit, observable acceptance criteria.
3. **Explore:** use focused read-only agents for complex repository/data-flow discovery. Require concise evidence and paths.
4. **Plan:** produce one bounded implementation and verification plan.
5. **Approval:** in feature mode, wait for explicit approval before application edits. Skip only when the user already approved the described edits. Verify-only mode does not edit.
6. **Implement:** assign non-overlapping ownership, make the smallest complete change, and preserve manual work.
7. **Focused tests:** run the narrowest relevant static and automated checks; add tests for changed behavior.
8. **Full verification:** run `./scripts/verify.sh` and inspect its report. Mark non-applicable or blocked checks honestly.
9. **Playwright inspection:** verify affected real-browser behavior; inspect console errors, failed application requests, screenshots, traces, and service logs. Use interactive browser automation for exploratory QA when available, but keep repeatable assertions in Playwright tests.
10. **Repair loop:** capture the failure and exit code, assign focused investigation, apply the smallest permitted fix, rerun the narrow check, then full verification. Stop after five full cycles or a genuine blocker.
11. **Independent review:** use a read-only reviewer on the complete working-tree diff. Resolve material findings and rerun affected checks.
12. **Final report:** state `PASS`, `PARTIAL`, or `FAIL`; list executed commands and exit codes, browser evidence, blockers, risks, and pre-existing versus current-task changes.

For manual work, use `./scripts/verify-changes.sh`. Never infer authorship from Git status, never reset or revert user changes, and never turn an unexecuted required check into a pass.
