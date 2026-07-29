#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
RUNTIME=${COACH_RUNTIME_DIR:-$ROOT/artifacts/runtime/manual}
LOGS="$ROOT/artifacts/service-logs"
mkdir -p "$RUNTIME" "$LOGS"
CLEANUP_RUNTIME="$RUNTIME/startup-$$"
mkdir -p "$CLEANUP_RUNTIME"
complete=false
api_pid_started=false
frontend_pid_started=false
api_compose_started=false

cleanup_partial() {
  if [ "$complete" != true ]; then
    env COACH_RUNTIME_DIR="$CLEANUP_RUNTIME" "$ROOT/scripts/dev-stop.sh" >/dev/null 2>&1 || true
    if [ "$api_pid_started" = true ]; then rm -f "$RUNTIME/api.pid"; fi
    if [ "$frontend_pid_started" = true ]; then rm -f "$RUNTIME/frontend.pid"; fi
    if [ "$api_compose_started" = true ]; then rm -f "$RUNTIME/api.compose-started"; fi
  else
    rm -f "$CLEANUP_RUNTIME/api.pid" "$CLEANUP_RUNTIME/frontend.pid" "$CLEANUP_RUNTIME/api.compose-started"
  fi
  rmdir "$CLEANUP_RUNTIME" 2>/dev/null || true
}
trap cleanup_partial 0 HUP INT TERM

pid_file_is_live() {
  file=$1
  [ -f "$file" ] || return 1
  pid=$(sed -n '1p' "$file")
  case "$pid" in
    ''|*[!0-9]*) return 1 ;;
    *) kill -0 "$pid" 2>/dev/null ;;
  esac
}

reconcile_pid_file() {
  file=$1
  label=$2
  if [ -f "$file" ] && ! pid_file_is_live "$file"; then
    echo "Removing stale $label ownership marker: $file" >&2
    rm -f "$file"
  fi
}

reconcile_pid_file "$RUNTIME/api.pid" API
reconcile_pid_file "$RUNTIME/frontend.pid" frontend
if [ -f "$RUNTIME/api.compose-started" ]; then
  expected_container=$(sed -n '1p' "$RUNTIME/api.compose-started")
  if [ -z "$expected_container" ]; then
    echo "Removing stale API Compose ownership marker: $RUNTIME/api.compose-started" >&2
    rm -f "$RUNTIME/api.compose-started"
  elif current_container=$(cd "$ROOT" && docker compose ps -q api 2>/dev/null); then
    if [ "$current_container" != "$expected_container" ]; then
      echo "Removing stale API Compose ownership marker: $RUNTIME/api.compose-started" >&2
      rm -f "$RUNTIME/api.compose-started"
    fi
  else
    echo "Unable to verify API Compose ownership marker; preserving it." >&2
  fi
fi

if [ "${COACH_RUNTIME_RECONCILE_ONLY:-false}" = true ]; then
  complete=true
  cleanup_partial
  trap - 0 HUP INT TERM
  exit 0
fi

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

api_is_coach() {
  response=$(curl -fsS http://127.0.0.1:8080/readyz 2>/dev/null) || return 1
  case "$response" in
    *'"service":"personalized-ai-coach-api"'*) return 0 ;;
    *) return 1 ;;
  esac
}

frontend_is_coach() {
  response=$(curl -fsS http://127.0.0.1:5173 2>/dev/null) || return 1
  case "$response" in
    *'<title>Nora — AI Learning Coach</title>'*) return 0 ;;
    *) return 1 ;;
  esac
}

if api_is_coach; then
  echo "Reusing API already listening on http://127.0.0.1:8080"
else
  if curl -fsS http://127.0.0.1:8080/readyz >/dev/null 2>&1; then
    echo "Port 8080 is occupied by an unidentified service; refusing to reuse it." >&2
    exit 1
  fi
  if [ -f "$RUNTIME/api.pid" ] || [ -f "$RUNTIME/api.compose-started" ]; then
    echo "An owned API process exists but is not ready; run scripts/dev-stop.sh before restarting." >&2
    exit 1
  fi
  if command -v go >/dev/null 2>&1; then
    (cd "$ROOT/backend" && exec env APP_ENV=development go run ./cmd/api) >"$LOGS/api.log" 2>&1 &
    api_pid=$!
    echo "$api_pid" >"$RUNTIME/api.pid"
    echo "$api_pid" >"$CLEANUP_RUNTIME/api.pid"
    api_pid_started=true
  elif command -v docker >/dev/null 2>&1; then
    (cd "$ROOT" && docker compose up -d --build api) >"$LOGS/api-compose.log" 2>&1
    api_container=$(cd "$ROOT" && docker compose ps -q api)
    echo "$api_container" >"$RUNTIME/api.compose-started"
    echo "$api_container" >"$CLEANUP_RUNTIME/api.compose-started"
    api_compose_started=true
  else
    echo "API startup requires Go 1.22+ or Docker." >&2
    exit 127
  fi
  wait_for API http://127.0.0.1:8080/readyz 120
  if ! api_is_coach; then
    echo "The service started on port 8080 does not identify as Personalized AI Coach." >&2
    exit 1
  fi
fi

if frontend_is_coach; then
  echo "Reusing frontend already listening on http://127.0.0.1:5173"
else
  if curl -fsS http://127.0.0.1:5173 >/dev/null 2>&1; then
    echo "Port 5173 is occupied by an unidentified service; refusing to reuse it." >&2
    exit 1
  fi
  if [ -f "$RUNTIME/frontend.pid" ]; then
    echo "An owned frontend process exists but is not ready; run scripts/dev-stop.sh before restarting." >&2
    exit 1
  fi
  (cd "$ROOT" && exec env VITE_DEV_TOKEN=dev:verification pnpm --filter frontend dev --host 127.0.0.1) >"$LOGS/frontend.log" 2>&1 &
  frontend_pid=$!
  echo "$frontend_pid" >"$RUNTIME/frontend.pid"
  echo "$frontend_pid" >"$CLEANUP_RUNTIME/frontend.pid"
  frontend_pid_started=true
  wait_for frontend http://127.0.0.1:5173 120
  if ! frontend_is_coach; then
    echo "The service started on port 5173 does not identify as Personalized AI Coach." >&2
    exit 1
  fi
fi

echo "Frontend: http://127.0.0.1:5173"
echo "API: http://127.0.0.1:8080"
complete=true
cleanup_partial
trap - 0 HUP INT TERM
