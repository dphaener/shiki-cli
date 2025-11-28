// Package components provides specialized UI components for the TUI.
// ResourceMatcher handles intelligent matching of tool resources to determine rollup behavior.
package components

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ResourceMatcher provides intelligent resource classification and matching.
type ResourceMatcher struct {
	workingDir    string
	globPatterns  map[string]*regexp.Regexp
	operationMap  map[string][]string // Tool operations that work on similar resources
}

// NewResourceMatcher creates a new resource matcher with current working directory.
func NewResourceMatcher() *ResourceMatcher {
	wd, _ := os.Getwd()

	return &ResourceMatcher{
		workingDir:   wd,
		globPatterns: make(map[string]*regexp.Regexp),
		operationMap: map[string][]string{
			// File operations that often work together
			"file_ops": {"Read", "Write", "Edit", "Grep", "Glob"},
			// Git operations
			"git_ops": {"git status", "git add", "git commit", "git push", "git diff"},
			// Build operations
			"build_ops": {"npm install", "npm build", "go build", "make", "cargo build"},
			// Test operations
			"test_ops": {"go test", "npm test", "pytest", "jest", "cargo test"},
		},
	}
}

// AreRelated determines if two resources should be considered related for rollup purposes.
func (rm *ResourceMatcher) AreRelated(resource1, resource2 string) bool {
	// Quick exact match check
	if resource1 == resource2 {
		return true
	}

	// Normalize and compare paths
	norm1 := rm.NormalizePath(resource1)
	norm2 := rm.NormalizePath(resource2)

	if norm1 == norm2 {
		return true
	}

	// Check if they're in the same directory
	if rm.sameDirectory(norm1, norm2) {
		return true
	}

	// Check if one is a parent/child relationship
	if rm.isParentChild(norm1, norm2) {
		return true
	}

	// Check for glob pattern relationships
	if rm.isGlobRelated(norm1, norm2) {
		return true
	}

	// Check for operation-based relationships (e.g., git commands)
	if rm.areRelatedOperations(resource1, resource2) {
		return true
	}

	return false
}

// NormalizePath converts relative paths to absolute and cleans them.
func (rm *ResourceMatcher) NormalizePath(path string) string {
	if path == "" {
		return path
	}

	// Handle special tool-specific patterns
	if rm.isToolSpecificResource(path) {
		return path // Keep tool names as-is
	}

	// Handle URLs
	if rm.isURL(path) {
		return path
	}

	// Handle shell commands
	if rm.isShellCommand(path) {
		return rm.normalizeShellCommand(path)
	}

	// Handle file paths
	if rm.isFilePath(path) {
		return rm.normalizeFilePath(path)
	}

	// Default: return as-is
	return path
}

// sameDirectory checks if two normalized paths are in the same directory.
func (rm *ResourceMatcher) sameDirectory(path1, path2 string) bool {
	if !rm.isFilePath(path1) || !rm.isFilePath(path2) {
		return false
	}

	dir1 := filepath.Dir(path1)
	dir2 := filepath.Dir(path2)

	return dir1 == dir2
}

// isParentChild checks if one path is a parent or child of another.
func (rm *ResourceMatcher) isParentChild(path1, path2 string) bool {
	if !rm.isFilePath(path1) || !rm.isFilePath(path2) {
		return false
	}

	// Check if path1 is a parent of path2
	rel1, err1 := filepath.Rel(path1, path2)
	if err1 == nil && !strings.HasPrefix(rel1, "..") {
		return true
	}

	// Check if path2 is a parent of path1
	rel2, err2 := filepath.Rel(path2, path1)
	if err2 == nil && !strings.HasPrefix(rel2, "..") {
		return true
	}

	return false
}

// isGlobRelated checks if paths might be related via glob patterns.
func (rm *ResourceMatcher) isGlobRelated(path1, path2 string) bool {
	// Check if either path is a glob pattern
	if strings.Contains(path1, "*") {
		return rm.matchesGlob(path1, path2)
	}

	if strings.Contains(path2, "*") {
		return rm.matchesGlob(path2, path1)
	}

	// Check if they share a common glob-like base
	return rm.shareCommonGlobBase(path1, path2)
}

// areRelatedOperations checks if two operation names are conceptually related.
func (rm *ResourceMatcher) areRelatedOperations(op1, op2 string) bool {
	for _, operations := range rm.operationMap {
		hasOp1 := rm.containsOperation(operations, op1)
		hasOp2 := rm.containsOperation(operations, op2)

		if hasOp1 && hasOp2 {
			return true
		}
	}
	return false
}

// isToolSpecificResource checks if this is a tool name rather than a resource.
func (rm *ResourceMatcher) isToolSpecificResource(resource string) bool {
	// Common tool names that aren't file paths
	toolNames := []string{
		"Bash", "WebFetch", "WebSearch", "Task", "Skill",
		"Read", "Write", "Edit", "Glob", "Grep",
	}

	for _, tool := range toolNames {
		if strings.EqualFold(resource, tool) {
			return true
		}
	}
	return false
}

// isURL checks if the resource is a URL.
func (rm *ResourceMatcher) isURL(resource string) bool {
	return strings.HasPrefix(resource, "http://") ||
		   strings.HasPrefix(resource, "https://") ||
		   strings.HasPrefix(resource, "ftp://")
}

// isShellCommand checks if the resource is a shell command.
func (rm *ResourceMatcher) isShellCommand(resource string) bool {
	// Look for command patterns
	if strings.Contains(resource, " ") {
		// Likely a command with arguments
		parts := strings.Fields(resource)
		if len(parts) > 0 {
			// Check if first part looks like a command
			cmd := parts[0]
			return !rm.isFilePath(cmd) && !strings.Contains(cmd, "/")
		}
	}
	return false
}

// isFilePath checks if the resource looks like a file path.
func (rm *ResourceMatcher) isFilePath(resource string) bool {
	// Check for obvious file path indicators
	if strings.HasPrefix(resource, "/") ||
	   strings.HasPrefix(resource, "./") ||
	   strings.HasPrefix(resource, "../") ||
	   strings.Contains(resource, "/") {
		return true
	}

	// Check for file extensions
	if strings.Contains(resource, ".") {
		ext := filepath.Ext(resource)
		if len(ext) > 1 && len(ext) <= 5 { // Reasonable extension length
			return true
		}
	}

	return false
}

// normalizeFilePath normalizes file paths to absolute form.
func (rm *ResourceMatcher) normalizeFilePath(path string) string {
	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		// Fallback to manual resolution
		if strings.HasPrefix(path, "./") {
			absPath = filepath.Join(rm.workingDir, path[2:])
		} else if strings.HasPrefix(path, "../") {
			absPath = filepath.Join(rm.workingDir, path)
		} else if !filepath.IsAbs(path) {
			absPath = filepath.Join(rm.workingDir, path)
		} else {
			absPath = path
		}
	}

	// Clean the path
	return filepath.Clean(absPath)
}

// normalizeShellCommand normalizes shell commands for comparison.
func (rm *ResourceMatcher) normalizeShellCommand(command string) string {
	// Extract the base command and primary arguments
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return command
	}

	baseCmd := parts[0]

	// Normalize common command patterns
	switch baseCmd {
	case "git":
		if len(parts) > 1 {
			return "git " + parts[1] // git status, git add, etc.
		}
	case "npm", "yarn":
		if len(parts) > 1 {
			return baseCmd + " " + parts[1] // npm install, npm build, etc.
		}
	case "go":
		if len(parts) > 1 {
			return "go " + parts[1] // go build, go test, etc.
		}
	case "cargo":
		if len(parts) > 1 {
			return "cargo " + parts[1] // cargo build, cargo test, etc.
		}
	}

	return baseCmd
}

// matchesGlob checks if a file path matches a glob pattern.
func (rm *ResourceMatcher) matchesGlob(pattern, path string) bool {
	matched, err := filepath.Match(pattern, path)
	if err != nil {
		return false
	}
	return matched
}

// shareCommonGlobBase checks if two paths share a common directory structure.
func (rm *ResourceMatcher) shareCommonGlobBase(path1, path2 string) bool {
	if !rm.isFilePath(path1) || !rm.isFilePath(path2) {
		return false
	}

	dir1 := filepath.Dir(path1)
	dir2 := filepath.Dir(path2)

	// Check if directories are the same or related
	if dir1 == dir2 {
		return true
	}

	// Check if one directory is a child of another
	return rm.isParentChild(dir1, dir2)
}

// containsOperation checks if an operation list contains a specific operation.
func (rm *ResourceMatcher) containsOperation(operations []string, target string) bool {
	for _, op := range operations {
		if strings.EqualFold(op, target) {
			return true
		}

		// Check for partial matches (e.g., "git status" contains "git")
		if strings.Contains(strings.ToLower(target), strings.ToLower(op)) {
			return true
		}
	}
	return false
}

// GetResourceType returns a classification of the resource type.
func (rm *ResourceMatcher) GetResourceType(resource string) string {
	if rm.isFilePath(resource) {
		return "file"
	}
	if rm.isURL(resource) {
		return "url"
	}
	if rm.isShellCommand(resource) {
		return "command"
	}
	if rm.isToolSpecificResource(resource) {
		return "tool"
	}
	return "other"
}

// GetDisplayName returns a user-friendly display name for a resource.
func (rm *ResourceMatcher) GetDisplayName(resource string) string {
	resourceType := rm.GetResourceType(resource)

	switch resourceType {
	case "file":
		// Show relative path if possible, otherwise filename
		if rel, err := filepath.Rel(rm.workingDir, resource); err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
		return filepath.Base(resource)

	case "command":
		// Show just the command name for shell commands
		parts := strings.Fields(resource)
		if len(parts) > 0 {
			return parts[0]
		}
		return resource

	default:
		return resource
	}
}