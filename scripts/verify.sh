#!/bin/sh
set -u

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
STAMP=$(date -u +%Y%m%dT%H%M%SZ)
RUN_DIR="$ROOT/artifacts/verification/$STAMP"
REPORT_DIR="$ROOT/artifacts/reports"
RESULTS="$RUN_DIR/results.tsv"
mkdir -p "$RUN_DIR/logs" "$REPORT_DIR"
: >"$RESULTS"

git -C "$ROOT" status --short >"$RUN_DIR/git-status-before.txt"
git -C "$ROOT" diff --name-only >"$RUN_DIR/tracked-changes-before.txt"
git -C "$ROOT" ls-files --others --exclude-standard >"$RUN_DIR/untracked-before.txt"

run_check() {
  name=$1
  shift
  log="$RUN_DIR/logs/$name.log"
  printf 'Running %s\n' "$name"
  "$@" >"$log" 2>&1
  code=$?
  if [ "$code" -eq 0 ]; then status=PASS; else status=FAIL; fi
  printf '%s\t%s\t%s\t%s\n' "$name" "$status" "$code" "$*" >>"$RESULTS"
  printf '%s: %s (exit %s)\n' "$name" "$status" "$code"
  LAST_CODE=$code
  if [ "$code" -eq 130 ] || [ "$code" -eq 143 ]; then
    exit "$code"
  fi
  return 0
}

blocked_check() {
  name=$1
  reason=$2
  printf '%s\tBLOCKED\t125\t%s\n' "$name" "$reason" >>"$RESULTS"
  printf '%s: BLOCKED (%s)\n' "$name" "$reason"
}

command_timeout() {
  limit=$1
  shift
  "$@" &
  command_pid=$!
  elapsed=0
  while kill -0 "$command_pid" 2>/dev/null; do
    if [ "$elapsed" -ge "$limit" ]; then
      kill "$command_pid" 2>/dev/null || true
      wait "$command_pid" 2>/dev/null || true
      return 124
    fi
    sleep 1
    elapsed=$((elapsed + 1))
  done
  wait "$command_pid"
}

backend_check() {
  if command -v go >/dev/null 2>&1; then
    (cd "$ROOT/backend" && "$@")
  elif command -v docker >/dev/null 2>&1; then
    docker run --rm -v "$ROOT/backend:/src" -w /src golang:1.22 "$@"
  else
    return 127
  fi
}

cleanup() {
  if [ "${SERVICES_STARTED:-false}" = true ]; then
    env COACH_RUNTIME_DIR="$RUN_DIR/runtime" "$ROOT/scripts/dev-stop.sh" >"$RUN_DIR/logs/service-cleanup.log" 2>&1 || true
  fi
}
trap cleanup EXIT HUP INT TERM

if [ "${PNPM_OFFLINE:-false}" = true ]; then
  run_check dependencies pnpm install --frozen-lockfile --offline --store-dir "$ROOT/.pnpm-store"
else
  run_check dependencies pnpm install --frozen-lockfile
fi
run_check frontend-format pnpm exec prettier --check "frontend/**/*.{ts,vue,json,css,md}"
run_check api-docs-format pnpm exec prettier --check "api/**/*.{yaml,yml}" "docs/**/*.md"
run_check frontend-typecheck pnpm --filter frontend typecheck
run_check frontend-lint pnpm --filter frontend lint
run_check frontend-build pnpm --filter frontend build
run_check frontend-tests pnpm --filter frontend test
run_check compose-config docker compose -f "$ROOT/docker-compose.yml" config --quiet
run_check openapi pnpm --config.ignore-scripts=true --package=@redocly/cli@1.34.2 dlx redocly lint "$ROOT/api/openapi.yaml"
run_check asyncapi pnpm --config.ignore-scripts=true dlx @asyncapi/cli@3.4.0 validate "$ROOT/api/asyncapi.yaml"

if command -v kubectl >/dev/null 2>&1; then
  run_check kustomize kubectl kustomize "$ROOT/infrastructure/k8s/base"
else
  blocked_check kustomize "kubectl is unavailable"
fi
if command -v terraform >/dev/null 2>&1; then
  run_check terraform-init terraform -chdir="$ROOT/infrastructure/terraform" init -backend=false
  run_check terraform-validate terraform -chdir="$ROOT/infrastructure/terraform" validate
else
  blocked_check terraform-init "Terraform is unavailable"
  blocked_check terraform-validate "Terraform is unavailable"
fi

runtime_available=false
if command -v go >/dev/null 2>&1; then
  runtime_available=true
elif command -v docker >/dev/null 2>&1 && command_timeout 30 docker info >/dev/null 2>&1; then
  runtime_available=true
fi

if [ "$runtime_available" = true ]; then
  run_check backend-format backend_check sh -c 'test -z "$(gofmt -l .)"'
  run_check backend-vet backend_check go vet ./...
  run_check backend-build backend_check go build ./...
  run_check backend-tests backend_check go test ./...
  run_check backend-race-tests backend_check go test -race ./...
  run_check migration "$ROOT/scripts/verify-migration.sh"
  run_check service-start env COACH_RUNTIME_DIR="$RUN_DIR/runtime" "$ROOT/scripts/dev-start.sh"
  if [ "$LAST_CODE" -eq 0 ]; then
    if [ -f "$RUN_DIR/runtime/api.pid" ] || [ -f "$RUN_DIR/runtime/frontend.pid" ] || [ -f "$RUN_DIR/runtime/api.compose-started" ]; then
      SERVICES_STARTED=true
    fi
    run_check smoke "$ROOT/scripts/smoke-test.sh"
    run_check playwright env PLAYWRIGHT_OUTPUT_DIR="$RUN_DIR/playwright-results" PLAYWRIGHT_HTML_REPORT="$RUN_DIR/playwright-report" pnpm exec playwright test
  else
    blocked_check smoke "local services did not start"
    blocked_check playwright "local services did not start"
  fi
else
  blocked_check backend-format "Go is unavailable and Docker did not respond within 30 seconds"
  blocked_check backend-vet "Go is unavailable and Docker did not respond within 30 seconds"
  blocked_check backend-build "Go is unavailable and Docker did not respond within 30 seconds"
  blocked_check backend-tests "Go is unavailable and Docker did not respond within 30 seconds"
  blocked_check backend-race-tests "Go is unavailable and Docker did not respond within 30 seconds"
  blocked_check migration "Docker did not respond within 30 seconds"
  blocked_check service-start "Go is unavailable and Docker did not respond within 30 seconds"
  blocked_check smoke "required API service is unavailable"
  blocked_check playwright "required API service is unavailable"
fi

git -C "$ROOT" status --short >"$RUN_DIR/git-status-after.txt"
git -C "$ROOT" diff --name-only >"$RUN_DIR/tracked-changes-after.txt"
git -C "$ROOT" ls-files --others --exclude-standard >"$RUN_DIR/untracked-after.txt"
comm -13 "$RUN_DIR/tracked-changes-before.txt" "$RUN_DIR/tracked-changes-after.txt" >"$RUN_DIR/new-tracked-changes.txt"
comm -13 "$RUN_DIR/untracked-before.txt" "$RUN_DIR/untracked-after.txt" >"$RUN_DIR/new-untracked-files.txt"

failed=$(awk -F '\t' '$2 == "FAIL" { count++ } END { print count + 0 }' "$RESULTS")
blocked=$(awk -F '\t' '$2 == "BLOCKED" { count++ } END { print count + 0 }' "$RESULTS")
if [ "$failed" -gt 0 ]; then final=FAIL; elif [ "$blocked" -gt 0 ]; then final=PARTIAL; else final=PASS; fi
branch=$(git -C "$ROOT" branch --show-current)
commit=$(git -C "$ROOT" rev-parse HEAD)
playwright_state=$(awk -F '\t' '$1 == "playwright" { print $2 }' "$RESULTS")

{
  echo "# Verification report"
  echo
  echo "- Timestamp: $STAMP"
  echo "- Branch: $branch"
  echo "- Baseline commit: $commit"
  echo "- Final status: **$final**"
  echo "- Frontend URL: http://127.0.0.1:5173"
  echo "- API URL: http://127.0.0.1:8080"
  echo
  echo "## Checks"
  echo
  echo '| Check | Result | Exit | Command |'
  echo '|---|---:|---:|---|'
  awk -F '\t' '{ printf "| `%s` | %s | %s | `%s` |\n", $1, $2, $3, $4 }' "$RESULTS"
  echo
  echo "## Browser and service evidence"
  echo
  if [ "$playwright_state" = PASS ]; then
    echo "- Browser scenario: loaded the authenticated daily session and navigated to its lesson."
    echo "- Console errors and failed requests: none; asserted by Playwright and attached as browser diagnostics."
  else
    echo "- Browser scenario: $playwright_state — the real-API scenario did not complete in this run."
    echo "- Console errors and failed requests: not claimed for the blocked/failed real-API scenario."
  fi
  if [ "$playwright_state" = PASS ] || [ "$playwright_state" = FAIL ]; then
    echo "- Playwright report: artifacts/verification/$STAMP/playwright-report/index.html"
    echo "- Playwright results: artifacts/verification/$STAMP/playwright-results/"
  else
    echo "- Playwright report/results: not produced by this run"
  fi
  echo "- Service logs: artifacts/service-logs/"
  echo "- Check logs: artifacts/verification/$STAMP/logs/"
  echo
  echo "## Changed files"
  echo
  echo "Pre-run Git status: artifacts/verification/$STAMP/git-status-before.txt"
  echo "Post-run Git status: artifacts/verification/$STAMP/git-status-after.txt"
  echo "Files first appearing during this run: artifacts/verification/$STAMP/new-tracked-changes.txt and new-untracked-files.txt"
  echo
  echo "## Applicability"
  echo
  echo "- Integration tests: NOT_APPLICABLE — no separate integration-test command exists; API behavior is covered by Go router tests and the smoke/E2E checks."
  echo "- External LLM/cloud services: NOT_APPLICABLE — local development intentionally uses the deterministic in-memory/fake-model profile."
  echo "- Interactive browser QA and independent review are agent workflow gates, not shell-script gates."
  echo
  echo "## Remaining risks"
  echo
  echo "Review failed check logs above. A failed required check makes this report FAIL."
} >"$REPORT_DIR/latest-verification.md"

node -e 'const fs=require("fs"); const [input,output,status,stamp]=process.argv.slice(1); const checks=fs.readFileSync(input,"utf8").trim().split("\n").filter(Boolean).map(line=>{const [name,result,exitCode,command]=line.split("\t");return{name,result,exitCode:Number(exitCode),command}}); fs.writeFileSync(output,JSON.stringify({timestamp:stamp,finalStatus:status,checks},null,2)+"\n")' "$RESULTS" "$REPORT_DIR/latest-verification.json" "$final" "$STAMP"

cp "$REPORT_DIR/latest-verification.md" "$RUN_DIR/report.md"
printf 'Verification result: %s\nReport: %s\n' "$final" "$REPORT_DIR/latest-verification.md"
if [ "$final" = PASS ]; then exit 0; else exit 1; fi
