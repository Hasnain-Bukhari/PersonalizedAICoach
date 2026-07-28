#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
RUNTIME=${COACH_RUNTIME_DIR:-$ROOT/artifacts/runtime/manual}

stop_pid() {
  file=$1
  label=$2
  if [ ! -f "$file" ]; then
    return
  fi
  pid=$(sed -n '1p' "$file")
  case "$pid" in
    ''|*[!0-9]*) echo "Ignoring invalid $label PID file: $file" >&2 ;;
    *)
      if kill -0 "$pid" 2>/dev/null; then
        kill "$pid"
        count=20
        while kill -0 "$pid" 2>/dev/null && [ "$count" -gt 0 ]; do
          count=$((count - 1))
          sleep 1
        done
      fi
      ;;
  esac
  rm -f "$file"
}

stop_pid "$RUNTIME/frontend.pid" frontend
stop_pid "$RUNTIME/api.pid" API

if [ -f "$RUNTIME/api.compose-started" ]; then
  expected_container=$(sed -n '1p' "$RUNTIME/api.compose-started")
  current_container=$(cd "$ROOT" && docker compose ps -q api 2>/dev/null || true)
  if [ -n "$expected_container" ] && [ "$current_container" = "$expected_container" ]; then
    (cd "$ROOT" && docker compose stop api)
  fi
  rm -f "$RUNTIME/api.compose-started"
fi
