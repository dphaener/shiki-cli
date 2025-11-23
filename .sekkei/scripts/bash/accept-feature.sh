#!/usr/bin/env bash
set -euo pipefail

if ! command -v ruby >/dev/null 2>&1; then
  echo "Error: ruby is required but was not found on PATH." >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUBY_HELPER="$SCRIPT_DIR/tasks-cli.rb"

if [[ ! -f "$RUBY_HELPER" ]]; then
  echo "Error: tasks-cli helper not found at $RUBY_HELPER" >&2
  exit 1
fi

ruby "$RUBY_HELPER" accept "$@"
