#!/usr/bin/env bash
set -euo pipefail

if ! command -v ruby >/dev/null 2>&1; then
  echo "Error: ruby is required but was not found on PATH." >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMMON_SH="$SCRIPT_DIR/../common.sh"
if [[ -f "$COMMON_SH" ]]; then
  # shellcheck source=/dev/null
  source "$COMMON_SH"

  if [[ -z "${SEKKEI_AUTORETRY:-}" ]]; then
    repo_root=$(get_repo_root)
    current_branch=$(get_current_branch)
    if [[ ! "$current_branch" =~ ^[0-9]{3}- ]]; then
      if latest_worktree=$(find_latest_feature_worktree "$repo_root" 2>/dev/null); then
        if [[ -d "$latest_worktree" ]]; then
          >&2 echo "[sekkei] Auto-running merge inside $latest_worktree (current branch: $current_branch)"
          (
            cd "$latest_worktree" && \
            SEKKEI_AUTORETRY=1 "$0" "$@"
          )
          exit $?
        fi
      fi
    fi
  fi
fi

RUBY_HELPER="$SCRIPT_DIR/tasks-cli.rb"

if [[ ! -f "$RUBY_HELPER" ]]; then
  echo "Error: tasks-cli helper not found at $RUBY_HELPER" >&2
  exit 1
fi

ruby "$RUBY_HELPER" merge "$@"
