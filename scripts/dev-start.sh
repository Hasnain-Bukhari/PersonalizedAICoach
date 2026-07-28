#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
RUNTIME=${COACH_RUNTIME_DIR:-$ROOT/artifacts/runtime/manual}
LOGS="$ROOT/artifacts/service-logs"
mkdir -p "$RUNTIME" "$LOGS"
rm -f "$RUNTIME/api.pid" "$RUNTIME/frontend.pid" "$RUNTIME/api.compose-started"
complete=false
trap 'if [ "$complete" != true ]; then "$ROOT/scripts/dev-stop.sh" >/dev/null 2>&1 || true; fi' 0 HUP INT TERM

wait_for() {
  name=$1
  url=$2
  attempts=${3:-60}
  while [ "$attempts" -gt 0 ]; do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    attempts=$((attempts - 1))
    sleep 1
  done
  echo "$name did not become ready at $url" >&2
  return 1
}

if curl -fsS http://127.0.0.1:8080/readyz >/dev/null 2>&1; then
  echo "Reusing API already listening on http://127.0.0.1:8080"
else
  if command -v go >/dev/null 2>&1; then
    (cd "$ROOT/backend" && exec env APP_ENV=development go run ./cmd/api) >"$LOGS/api.log" 2>&1 &
    echo $! >"$RUNTIME/api.pid"
  elif command -v docker >/dev/null 2>&1; then
    (cd "$ROOT" && docker compose up -d --build api) >"$LOGS/api-compose.log" 2>&1
    docker compose ps -q api >"$RUNTIME/api.compose-started"
  else
    echo "API startup requires Go 1.22+ or Docker." >&2
    exit 127
  fi
  wait_for API http://127.0.0.1:8080/readyz 120
fi

if curl -fsS http://127.0.0.1:5173 >/dev/null 2>&1; then
  echo "Reusing frontend already listening on http://127.0.0.1:5173"
else
  (cd "$ROOT" && exec env VITE_DEV_TOKEN=dev:verification pnpm --filter frontend dev --host 127.0.0.1) >"$LOGS/frontend.log" 2>&1 &
  echo $! >"$RUNTIME/frontend.pid"
  wait_for frontend http://127.0.0.1:5173 120
fi

echo "Frontend: http://127.0.0.1:5173"
echo "API: http://127.0.0.1:8080"
complete=true
trap - 0 HUP INT TERM
