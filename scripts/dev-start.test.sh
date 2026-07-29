#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
TEST_RUNTIME=$(mktemp -d)
owned_pid=""

cleanup() {
  if [ -n "$owned_pid" ] && kill -0 "$owned_pid" 2>/dev/null; then
    kill "$owned_pid" 2>/dev/null || true
  fi
  rm -f "$TEST_RUNTIME/api.pid" "$TEST_RUNTIME/frontend.pid" "$TEST_RUNTIME/api.compose-started"
  rmdir "$TEST_RUNTIME" 2>/dev/null || true
}
trap cleanup EXIT HUP INT TERM

sh -c 'trap "exit 0" TERM; while :; do sleep 1; done' &
owned_pid=$!
echo "$owned_pid" >"$TEST_RUNTIME/api.pid"
echo "not-a-pid" >"$TEST_RUNTIME/frontend.pid"
: >"$TEST_RUNTIME/api.compose-started"

env COACH_RUNTIME_DIR="$TEST_RUNTIME" COACH_RUNTIME_RECONCILE_ONLY=true "$ROOT/scripts/dev-start.sh"

test "$(sed -n '1p' "$TEST_RUNTIME/api.pid")" = "$owned_pid"
test ! -e "$TEST_RUNTIME/frontend.pid"
test ! -e "$TEST_RUNTIME/api.compose-started"
printf 'Runtime ownership marker reconciliation passed.\n'
