---
description: Start, stop, and manage the Sekkei dashboard as a background daemon
scripts:
  sh: .sekkei/scripts/bash/setup-dashboard.sh "{ARGS}"
  ps: .sekkei/scripts/powershell/Setup-Dashboard.ps1 "{ARGS}"
---

**Path**: [templates/commands/dashboard.md](templates/commands/dashboard.md)

## Usage

```
/sekkei.dashboard [--port PORT] [--kill]
```

Start, stop, or manage the Sekkei dashboard web interface.

**Path reference rule:** When you mention directories or files, provide either the absolute path or a path relative to the project root (for example, `sekkei-specs/<feature>/tasks/`). Never refer to a folder by name alone.

## Arguments

- `--port PORT` - Optional port number (1-65535) for the dashboard server
- `--kill` - Stop the dashboard daemon for this project

## Outline

### Dashboard Access

The dashboard shows ALL features across the project and runs from the **main repository**, not from individual feature worktrees.

### Important: Worktree Handling

**If you're in a feature worktree**, the dashboard file is in the main repo, not in your worktree.

The dashboard is project-wide (shows all features), so it must be accessed from the main repository location.

### Implementation Steps

1. **Check Current Location**:
   - Determine if running from a feature worktree or main repository
   - If in worktree, identify the main repository path using `git worktree list`
   - Notify user if dashboard will run from a different location

2. **Handle Kill Flag**:
   - If `--kill` flag is provided, stop the dashboard daemon
   - Use the appropriate script based on platform (bash or PowerShell)
   - Display success/failure message
   - Exit

3. **Validate Port (if provided)**:
   - Check port is between 1 and 65535
   - If invalid, display error and exit

4. **Start/Access Dashboard**:
   - Run the appropriate dashboard script from the main repository
   - Script will:
     - Check if dashboard is already running for this project
     - Start new instance if not running
     - Use preferred port if available, fallback to next available port
     - Return dashboard URL and status

5. **Display Dashboard Information**:
   ```
   Sekkei Dashboard
   ============================================================

     Project Root: /path/to/project
     URL: http://localhost:PORT
     Port: PORT

     ✅ Status: Started new dashboard instance on port PORT
        (or "Dashboard already running on port PORT")

   ============================================================
   ```

6. **Open Browser**:
   - Attempt to open dashboard URL in default browser
   - If successful: Display "✅ Opening dashboard in your browser..."
   - If failed: Display manual URL with instructions

### Success Criteria

- User sees the dashboard URL clearly displayed
- Browser opens automatically to the dashboard (when possible)
- If browser doesn't open, user gets clear instructions with URL
- Dashboard runs as background daemon
- Multiple projects can run dashboards simultaneously on different ports
- Port conflicts are handled gracefully with automatic port selection
- Stopping dashboard (`--kill`) cleanly shuts down the daemon
- Error messages are helpful and actionable

### Error Handling

- **Invalid port**: Display clear error message with valid range
- **Dashboard startup failure**: Provide troubleshooting steps
- **Not a Sekkei project**: Suggest running `sekkei init`
- **Git not available**: Fallback to current directory, warn user
- **Permission errors**: Display clear message about file/port permissions
