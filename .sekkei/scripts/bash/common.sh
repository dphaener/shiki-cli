#!/usr/bin/env bash
# Common functions and variables for all scripts

# Get repository root, with fallback for non-git repositories
get_repo_root() {
    if git rev-parse --show-toplevel >/dev/null 2>&1; then
        git rev-parse --show-toplevel
    else
        # Fall back to script location for non-git repos
        local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
        (cd "$script_dir/../../.." && pwd)
    fi
}

# Get current branch, with fallback for non-git repositories
get_current_branch() {
    # First check if SPECIFY_FEATURE environment variable is set
    if [[ -n "${SPECIFY_FEATURE:-}" ]]; then
        echo "$SPECIFY_FEATURE"
        return
    fi
    
    # Then check git if available
    if git rev-parse --abbrev-ref HEAD >/dev/null 2>&1; then
        git rev-parse --abbrev-ref HEAD
        return
    fi
    
    # For non-git repos, try to find the latest feature directory
    local repo_root=$(get_repo_root)
    local specs_dir="$repo_root/sekkei-specs"
    
    if [[ -d "$specs_dir" ]]; then
        local latest_feature=""
        local highest=0
        
        for dir in "$specs_dir"/*; do
            if [[ -d "$dir" ]]; then
                local dirname=$(basename "$dir")
                if [[ "$dirname" =~ ^([0-9]{3})- ]]; then
                    local number=${BASH_REMATCH[1]}
                    number=$((10#$number))
                    if [[ "$number" -gt "$highest" ]]; then
                        highest=$number
                        latest_feature=$dirname
                    fi
                fi
            fi
        done
        
        if [[ -n "$latest_feature" ]]; then
            echo "$latest_feature"
            return
        fi
    fi
    
    echo "main"  # Final fallback
}

# Check if we have git available
has_git() {
    git rev-parse --show-toplevel >/dev/null 2>&1
}

check_feature_branch() {
    local branch="$1"
    local has_git_repo="$2"

    # For non-git repos, we can't enforce branch naming but still provide output
    if [[ "$has_git_repo" != "true" ]]; then
        echo "[sekkei] Warning: Git repository not detected; skipped branch validation" >&2
        return 0
    fi

    if [[ ! "$branch" =~ ^[0-9]{3}- ]]; then
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" >&2
        echo "❌ ERROR: Command run from wrong location!" >&2
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" >&2
        echo "" >&2
        echo "Current location: $(pwd)" >&2
        echo "Current branch: $branch" >&2
        echo "Required: Feature branch (e.g., 001-feature-name)" >&2
        echo "" >&2

        # Help agents find the worktree by checking if any exist
        local repo_root=$(get_repo_root)
        if [[ -d "$repo_root/.worktrees" ]] && [[ -n "$(ls -A "$repo_root/.worktrees" 2>/dev/null)" ]]; then
            echo "🔧 TO FIX THIS ISSUE:" >&2
            echo "" >&2
            echo "1. List available worktrees:" >&2
            echo "   ls .worktrees/" >&2
            echo "" >&2
            echo "2. Navigate to a worktree:" >&2
            local first_worktree=$(ls -1 "$repo_root/.worktrees" 2>/dev/null | head -1)
            if [[ -n "$first_worktree" ]]; then
                echo "   cd .worktrees/$first_worktree" >&2
            else
                echo "   cd .worktrees/<feature-name>" >&2
            fi
            echo "" >&2
            echo "3. Retry the command" >&2
            echo "" >&2
            echo "Available worktrees:" >&2
            ls -1 "$repo_root/.worktrees" 2>/dev/null | sed 's/^/  ✓ /' >&2
        else
            echo "🔧 TO FIX THIS ISSUE:" >&2
            echo "" >&2
            echo "No worktrees found. Create a new feature with:" >&2
            echo "  sekkei specify" >&2
            echo "" >&2
            echo "Or if you have a feature branch, create its worktree:" >&2
            echo "  git worktree add .worktrees/001-your-feature 001-your-feature" >&2
        fi
        echo "" >&2
        echo "💡 TIP: Run 'sekkei verify-setup' to diagnose issues" >&2
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" >&2

        return 1
    fi

    return 0
}

find_latest_feature_worktree() {
    local repo_root="$1"
    local worktrees_root="$repo_root/.worktrees"
    local latest_path=""
    local highest=-1

    if [[ ! -d "$worktrees_root" ]]; then
        return 1
    fi

    while IFS= read -r dir; do
        [[ -d "$dir" ]] || continue
        local base="$(basename "$dir")"
        if [[ "$base" =~ ^([0-9]{3})- ]]; then
            local number=$((10#${BASH_REMATCH[1]}))
            if (( number > highest )); then
                highest=$number
                latest_path="$dir"
            fi
        fi
    done < <(find "$worktrees_root" -mindepth 1 -maxdepth 1 -type d -print 2>/dev/null)

    if [[ -n "$latest_path" ]]; then
        printf '%s\n' "$latest_path"
        return 0
    fi

    return 1
}

get_feature_dir() { echo "$1/sekkei-specs/$2"; }

get_mission_exports() {
    local repo_root="$1"

    # Use Ruby for mission detection
    local ruby_bin="ruby"
    if ! command -v "$ruby_bin" >/dev/null 2>&1; then
        echo "[sekkei] Error: ruby interpreter not found; mission detection unavailable" >&2
        return 1
    fi

    # Get the directory where this script is located
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

    # Call the Ruby mission resolver script
    "$ruby_bin" "$script_dir/mission-resolver.rb" "$repo_root"
}

get_feature_paths() {
    local repo_root=$(get_repo_root)
    local current_branch=$(get_current_branch)
    local has_git_repo="false"
    
    if has_git; then
        has_git_repo="true"
    fi
    
    local feature_dir=$(get_feature_dir "$repo_root" "$current_branch")
    local mission_exports
    mission_exports=$(get_mission_exports "$repo_root") || return 1
    
    cat <<EOF
REPO_ROOT='$repo_root'
CURRENT_BRANCH='$current_branch'
HAS_GIT='$has_git_repo'
FEATURE_DIR='$feature_dir'
FEATURE_SPEC='$feature_dir/spec.md'
IMPL_PLAN='$feature_dir/plan.md'
TASKS='$feature_dir/tasks.md'
RESEARCH='$feature_dir/research.md'
DATA_MODEL='$feature_dir/data-model.md'
QUICKSTART='$feature_dir/quickstart.md'
CONTRACTS_DIR='$feature_dir/contracts'
EOF

    printf '%s\n' "$mission_exports"
}

check_file() { [[ -f "$1" ]] && echo "  ✓ $2" || echo "  ✗ $2"; }
check_dir() { [[ -d "$1" && -n $(ls -A "$1" 2>/dev/null) ]] && echo "  ✓ $2" || echo "  ✗ $2"; }
