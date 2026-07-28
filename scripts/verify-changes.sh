#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
echo "Verifying the complete working tree, including manual modified and untracked files:"
git -C "$ROOT" status --short
echo "No files will be reset, restored, or reverted."
exec "$ROOT/scripts/verify.sh"
