#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
CONTAINER="coach-migration-verify-$$"

cleanup() {
  docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
}
trap cleanup EXIT HUP INT TERM

docker run --rm -d --name "$CONTAINER" \
  -e POSTGRES_PASSWORD=verification_only \
  -e POSTGRES_DB=coach_verify \
  -v "$ROOT/backend/migrations:/migrations:ro" \
  pgvector/pgvector:pg16 >/dev/null

attempts=60
until docker exec "$CONTAINER" pg_isready -U postgres -d coach_verify >/dev/null 2>&1; do
  attempts=$((attempts - 1))
  if [ "$attempts" -le 0 ]; then
    echo "Disposable PostgreSQL did not become ready." >&2
    exit 1
  fi
  sleep 1
done

find "$ROOT/backend/migrations" -maxdepth 1 -type f -name '*.sql' ! -name '*.down.sql' | sort | while IFS= read -r migration; do
  docker exec "$CONTAINER" psql -v ON_ERROR_STOP=1 -U postgres -d coach_verify -f "/migrations/$(basename "$migration")"
done
find "$ROOT/backend/migrations" -maxdepth 1 -type f -name '*.down.sql' | sort -r | while IFS= read -r migration; do
  docker exec "$CONTAINER" psql -v ON_ERROR_STOP=1 -U postgres -d coach_verify -f "/migrations/$(basename "$migration")"
done
