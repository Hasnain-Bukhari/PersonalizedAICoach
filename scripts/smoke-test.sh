#!/bin/sh
set -eu

FRONTEND_URL=${FRONTEND_URL:-http://127.0.0.1:5173}
API_URL=${API_URL:-http://127.0.0.1:8080}

health=$(curl -fsS "$API_URL/readyz")
case "$health" in
  *'"status":"ok"'*) ;;
  *) echo "Unexpected API readiness response: $health" >&2; exit 1 ;;
esac

curl -fsS "$FRONTEND_URL" | grep -F '<div id="app"></div>' >/dev/null
curl -fsS -H 'Authorization: Bearer dev:smoke' \
  "$API_URL/api/v1/sessions/daily" | grep -F '"lesson"' >/dev/null

echo "API readiness, authenticated daily session, and frontend shell passed."
