#!/bin/sh
set -eu

if command -v go >/dev/null 2>&1; then
  cd backend
  exec env APP_ENV=development go run ./cmd/api
fi

if command -v docker >/dev/null 2>&1; then
  echo "Go is not installed; starting the API with Docker Compose."
  exec docker compose up --build api
fi

echo "Unable to start the API: install Go 1.22+ or Docker." >&2
exit 127
