#!/usr/bin/env bash
set -euo pipefail

# refresh-sekkei-tasks.sh
# Copy the latest task helper modules into an existing Sekkei project.
# Usage:
#   scripts/bash/refresh-sekkei-tasks.sh [<project-root>]
# If no project root is provided the script walks upward from the current
# directory until it finds a .sekkei/scripts directory.

usage() {
  cat <<'EOF'
Usage: refresh-sekkei-tasks.sh [<project-root>]

Copies the current repo's scripts/tasks helpers into the target project's
.sekkei/scripts/tasks directory. Provide the project root explicitly or run
from anywhere inside the project tree.
EOF
}

if [[ ${1:-} == "--help" ]] || [[ ${1:-} == "-h" ]]; then
  usage
  exit 0
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SOURCE_TASK_DIR="$REPO_ROOT/scripts/tasks"

if [[ ! -d "$SOURCE_TASK_DIR" ]]; then
  echo "Error: expected task helpers at $SOURCE_TASK_DIR" >&2
  exit 1
fi

if [[ $# -gt 1 ]]; then
  echo "Error: too many arguments" >&2
  usage >&2
  exit 1
fi

if [[ $# -eq 1 ]]; then
  if [[ ! -d $1 ]]; then
    echo "Error: project path '$1' does not exist or is not a directory" >&2
    exit 1
  fi
  START_PATH="$(cd "$1" && pwd)"
else
  START_PATH="$(pwd)"
fi

locate_project_root() {
  local current="$1"
  while true; do
    if [[ -d "$current/.sekkei/scripts" ]]; then
      echo "$current"
      return 0
    fi
    if [[ "$current" == "/" ]]; then
      break
    fi
    current="$(dirname "$current")"
  done
  return 1
}

PROJECT_ROOT="$(locate_project_root "$START_PATH" || true)"
if [[ -z "$PROJECT_ROOT" ]]; then
  echo "Error: unable to locate .sekkei/scripts starting from $START_PATH" >&2
  exit 1
fi

TARGET_SCRIPT_ROOT="$PROJECT_ROOT/.sekkei/scripts"
TARGET_TASK_DIR="$TARGET_SCRIPT_ROOT/tasks"

# Use Ruby for directory syncing
if ! command -v ruby >/dev/null 2>&1; then
    echo "Error: ruby interpreter not found; directory sync unavailable" >&2
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ruby "$SCRIPT_DIR/sync-directory.rb" "$SOURCE_TASK_DIR" "$TARGET_TASK_DIR"

echo "✅ Updated .sekkei scripts:"
echo "   Source : $SOURCE_TASK_DIR"
echo "   Target : $TARGET_TASK_DIR"
if [[ -f "$LEGACY_BACKUP" ]]; then
  echo "   Legacy backup saved at $LEGACY_BACKUP"
fi
