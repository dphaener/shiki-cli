package diff

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/pmezard/go-difflib/difflib"
)

// FileDiff represents a diff between two versions of a file
type FileDiff struct {
	FilePath    string `json:"file_path"`
	OldContent  string `json:"old_content"`
	NewContent  string `json:"new_content"`
	UnifiedDiff string `json:"unified_diff"`
	IsBinary    bool   `json:"is_binary"`
	IsLarge     bool   `json:"is_large"`
	TotalLines  int    `json:"total_lines"`
}

// Generator handles the creation of file diffs
type Generator struct {
	maxLines      int
	contextLines  int
	maxFileSize   int64 // Maximum file size in bytes to process
}

// NewGenerator creates a new diff generator with default settings
func NewGenerator() *Generator {
	return &Generator{
		maxLines:     100,  // Maximum lines to display in diff
		contextLines: 3,    // Context lines before and after changes
		maxFileSize:  1024 * 1024, // 1MB max file size
	}
}

// GenerateUnifiedDiff creates a unified diff between old and new file content
func (g *Generator) GenerateUnifiedDiff(oldContent, newContent, filePath string) (*FileDiff, error) {
	diff := &FileDiff{
		FilePath:   filePath,
		OldContent: oldContent,
		NewContent: newContent,
	}

	// Check if files are binary
	if g.IsBinaryContent([]byte(oldContent)) || g.IsBinaryContent([]byte(newContent)) {
		diff.IsBinary = true
		diff.UnifiedDiff = g.formatBinaryFileDiff(oldContent, newContent, filePath)
		return diff, nil
	}

	// Split content into lines for diffing
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	// Check if diff would be too large
	totalLines := len(oldLines) + len(newLines)
	diff.TotalLines = totalLines
	if totalLines > 1500 { // Mark as large if total lines > 1500
		diff.IsLarge = true
	}

	// Generate unified diff
	unifiedDiff := difflib.UnifiedDiff{
		A:        oldLines,
		B:        newLines,
		FromFile: "a/" + filepath.Base(filePath),
		ToFile:   "b/" + filepath.Base(filePath),
		Context:  g.contextLines,
	}

	diffText, err := difflib.GetUnifiedDiffString(unifiedDiff)
	if err != nil {
		return nil, fmt.Errorf("failed to generate unified diff: %w", err)
	}

	// Apply truncation if necessary
	diff.UnifiedDiff = g.truncateDiffIfNeeded(diffText)

	return diff, nil
}

// IsBinaryContent checks if content appears to be binary
func (g *Generator) IsBinaryContent(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	// Check for null bytes (common binary indicator)
	if bytes.Contains(content, []byte{0}) {
		return true
	}

	// Check if content is valid UTF-8 and contains reasonable text
	if !utf8.Valid(content) {
		return true
	}

	// Sample first 512 bytes to check for binary characteristics
	sampleSize := 512
	if len(content) < sampleSize {
		sampleSize = len(content)
	}

	sample := content[:sampleSize]
	nonPrintableCount := 0
	highAsciiCount := 0
	validUnicodeBytes := 0

	// Count UTF-8 runes to differentiate Unicode from binary data
	for len(sample) > 0 {
		r, size := utf8.DecodeRune(sample)
		if r == utf8.RuneError && size == 1 {
			// Invalid UTF-8 sequence
			break
		}

		if size > 1 {
			// Multi-byte UTF-8 sequence (valid Unicode)
			validUnicodeBytes += size
		} else if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			// Non-printable ASCII
			nonPrintableCount++
		} else if r >= 128 {
			// High ASCII byte (part of extended ASCII, not Unicode)
			highAsciiCount++
		}

		sample = sample[size:]
	}

	// If more than 30% of sampled bytes are non-printable, consider it binary
	nonPrintableThreshold := float64(sampleSize) * 0.3
	if float64(nonPrintableCount) > nonPrintableThreshold {
		return true
	}

	// If we have significant valid Unicode, it's text
	if validUnicodeBytes > 0 {
		return false
	}

	// If more than 50% of sampled bytes are high ASCII (not Unicode), likely binary
	highAsciiThreshold := float64(sampleSize) * 0.5
	if float64(highAsciiCount) > highAsciiThreshold {
		return true
	}

	return false
}

// formatBinaryFileDiff creates a summary for binary file changes
func (g *Generator) formatBinaryFileDiff(oldContent, newContent, filePath string) string {
	oldSize := len(oldContent)
	newSize := len(newContent)

	baseName := filepath.Base(filePath)

	if oldSize == 0 {
		return fmt.Sprintf("Binary file %s created (%d bytes)", baseName, newSize)
	}

	if newSize == 0 {
		return fmt.Sprintf("Binary file %s deleted (%d bytes)", baseName, oldSize)
	}

	return fmt.Sprintf("Binary file %s modified (%d → %d bytes)", baseName, oldSize, newSize)
}

// truncateDiffIfNeeded applies truncation to very large diffs
func (g *Generator) truncateDiffIfNeeded(diffText string) string {
	lines := strings.Split(diffText, "\n")

	if len(lines) <= g.maxLines {
		return diffText
	}

	// For large diffs, show first portion, omission notice, and last portion
	firstPortion := g.maxLines / 2
	lastPortion := g.maxLines / 2

	if firstPortion+lastPortion >= len(lines) {
		return diffText
	}

	var result []string

	// Add first portion
	result = append(result, lines[:firstPortion]...)

	// Add omission indicator
	omittedLines := len(lines) - firstPortion - lastPortion
	result = append(result, fmt.Sprintf("... %d lines omitted ...", omittedLines))

	// Add last portion
	if lastPortion > 0 {
		result = append(result, lines[len(lines)-lastPortion:]...)
	}

	return strings.Join(result, "\n")
}

// SetMaxLines configures the maximum number of diff lines to display
func (g *Generator) SetMaxLines(maxLines int) {
	if maxLines > 0 {
		g.maxLines = maxLines
	}
}

// SetContextLines configures the number of context lines around changes
func (g *Generator) SetContextLines(contextLines int) {
	if contextLines >= 0 {
		g.contextLines = contextLines
	}
}

// GetFileSize returns the size of content in bytes
func GetFileSize(content string) int64 {
	return int64(len([]byte(content)))
}

// SanitizeFilePath removes sensitive directory information from file paths
func SanitizeFilePath(filePath string) string {
	// Handle empty or special cases
	if filePath == "" {
		return "."
	}
	if filePath == "." || filePath == ".." {
		return filePath
	}
	if filePath == "/" {
		return "."
	}

	// Clean the path and remove control characters
	cleanPath := sanitizeControlCharacters(filePath)

	// Handle Windows-style paths specially on non-Windows systems
	// to avoid confusion with absolute path detection
	if filepath.Separator == '/' { // Unix-like system
		// Check if this looks like a Windows path
		if len(cleanPath) >= 2 && cleanPath[1] == ':' { // C:file or C:\...
			return cleanPath // Return as-is since it's not a real absolute path on Unix
		}
		if strings.HasPrefix(cleanPath, `\\`) { // UNC path
			return cleanPath // Return as-is since it's not a real absolute path on Unix
		}
	}

	// Convert to absolute path first to handle relative paths consistently
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		// If we can't get absolute path, clean the original and use basename
		return sanitizeControlCharacters(filepath.Base(cleanPath))
	}

	// Check for sensitive patterns and return basename if found
	sensitivePatterns := []string{
		"/secrets/",
		"/.env",
		"/credentials",
		"/keys/",
		"/.ssh/",
		"/tokens/",
		"/passwords/",
		"/run/secrets/",      // Docker secrets
		"/.aws/",             // AWS credentials
		"/service-account",   // GCP service account
		".pem",              // Certificate files
		".key",              // Private keys
		"/etc/shadow",       // System password files
		"/etc/passwd",       // System user files
		"/proc/",            // Process information
		"private.key",       // Private key files
		"/ssl/private/",     // SSL private directory
	}

	lowerPath := strings.ToLower(absPath)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerPath, pattern) {
			return sanitizeControlCharacters(filepath.Base(absPath))
		}
	}

	// Additional check for file extensions that are commonly sensitive
	ext := strings.ToLower(filepath.Ext(absPath))
	sensitiveExtensions := []string{".pem", ".key", ".p12", ".pfx", ".crt"}
	for _, sensExt := range sensitiveExtensions {
		if ext == sensExt {
			return sanitizeControlCharacters(filepath.Base(absPath))
		}
	}

	// For regular files, try to show a relative path if it's in current working directory
	// or subdirectory, otherwise show basename
	cwd, err := filepath.Abs(".")
	if err != nil {
		return sanitizeControlCharacters(filepath.Base(absPath))
	}

	rel, err := filepath.Rel(cwd, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		// If file is outside current directory, just show basename
		return sanitizeControlCharacters(filepath.Base(absPath))
	}

	// Show relative path if it's not too long
	if len(rel) > 60 {
		return sanitizeControlCharacters(filepath.Base(absPath))
	}

	return sanitizeControlCharacters(rel)
}

// sanitizeControlCharacters removes null bytes and control characters from strings
func sanitizeControlCharacters(s string) string {
	if s == "" {
		return s
	}

	var result strings.Builder
	result.Grow(len(s)) // Pre-allocate capacity

	for _, r := range s {
		// Skip null bytes and control characters (except tab)
		if r == 0 || (r < 32 && r != '\t') {
			continue
		}
		result.WriteRune(r)
	}

	cleaned := result.String()
	if cleaned == "" {
		return "sanitized_file"
	}

	return cleaned
}

// ValidateFileContent checks if file content is safe to display in diff
func ValidateFileContent(content string) bool {
	// Basic validation to avoid displaying potentially sensitive content
	lowerContent := strings.ToLower(content)

	// Check for common sensitive patterns in various formats
	sensitivePatterns := []string{
		// Credentials and passwords
		"password=", "password:", "pwd=", "passwd=",
		"auth_token=", "auth_token:", "authtoken=",
		"api_key=", "api_key:", "apikey=", "api-key=",
		"access_key=", "access_key:", "accesskey=",
		"secret_key=", "secret_key:", "secretkey=",
		"secret=", "secret:", "app_secret=",
		"token=", "bearer_token=", "bearer ",
		"private_key=", "privatekey=",

		// Cloud provider credentials
		"aws_access_key", "aws_secret_key", "aws_session_token",
		"azure_client_secret", "azure_tenant_id",
		"gcp_service_account", "google_application_credentials",
		"service_account_key",

		// Database credentials
		"db_password=", "database_password=", "mysql_password=",
		"postgres_password=", "mongodb_password=",
		"connection_string=", "jdbc:",

		// SSH and certificates
		"-----begin private key-----", "-----begin rsa private key-----",
		"-----begin openssh private key-----", "-----begin dsa private key-----",
		"-----begin ec private key-----", "-----begin encrypted private key-----",
		"-----begin certificate-----", "-----begin rsa certificate-----",
		"ssh-rsa ", "ssh-dss ", "ssh-ed25519 ",

		// Common secret formats
		"client_secret=", "app_key=", "encryption_key=",
		"signing_key=", "webhook_secret=", "hmac_secret=",
		"jwt_secret=", "session_secret=", "cookie_secret=",

		// Hardcoded credentials patterns (loose detection)
		"username:", "user:", "login:",
	}

	// Check for sensitive patterns
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerContent, pattern) {
			return false
		}
	}

	// Check for potential base64 encoded secrets (loose heuristic)
	if containsSuspiciousBase64(content) {
		return false
	}

	// Check for long hex strings that might be secrets
	if containsSuspiciousHexStrings(content) {
		return false
	}

	return true
}

// containsSuspiciousBase64 checks for potential base64 encoded secrets
func containsSuspiciousBase64(content string) bool {
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Skip comments and short lines
		if strings.HasPrefix(trimmedLine, "#") || strings.HasPrefix(trimmedLine, "//") || len(trimmedLine) < 32 {
			continue
		}

		// Look for assignment-like patterns with long base64-looking strings
		if strings.Contains(trimmedLine, "=") {
			parts := strings.Split(trimmedLine, "=")
			if len(parts) >= 2 {
				value := strings.TrimSpace(parts[1])
				// Remove quotes if present
				value = strings.Trim(value, `"'`)

				// Check if it looks like base64 and is long enough to be a secret
				if isLikelyBase64(value) && len(value) >= 32 {
					return true
				}
			}
		}
	}

	return false
}

// containsSuspiciousHexStrings checks for long hexadecimal strings that might be secrets
func containsSuspiciousHexStrings(content string) bool {
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Skip comments and short lines
		if strings.HasPrefix(trimmedLine, "#") || strings.HasPrefix(trimmedLine, "//") || len(trimmedLine) < 32 {
			continue
		}

		// Look for assignment-like patterns with long hex strings
		if strings.Contains(trimmedLine, "=") {
			parts := strings.Split(trimmedLine, "=")
			if len(parts) >= 2 {
				value := strings.TrimSpace(parts[1])
				// Remove quotes if present
				value = strings.Trim(value, `"'`)

				// Check if it's a long hex string (common for secrets)
				if isLikelyHex(value) && len(value) >= 32 {
					return true
				}
			}
		}
	}

	return false
}

// isLikelyBase64 checks if a string looks like base64
func isLikelyBase64(s string) bool {
	if len(s) < 4 {
		return false
	}

	// Base64 should be mostly alphanumeric with + / and =
	base64Chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
	validChars := 0

	for _, char := range s {
		if strings.ContainsRune(base64Chars, char) {
			validChars++
		}
	}

	// If more than 95% of characters are base64-valid, it's likely base64
	// This stricter threshold reduces false positives
	return float64(validChars)/float64(len(s)) > 0.95
}

// isLikelyHex checks if a string looks like hexadecimal
func isLikelyHex(s string) bool {
	if len(s) < 8 {
		return false
	}

	// Remove common hex prefixes
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")

	// Hex should be only 0-9, a-f, A-F
	hexChars := "0123456789abcdefABCDEF"
	validChars := 0

	for _, char := range s {
		if strings.ContainsRune(hexChars, char) {
			validChars++
		}
	}

	// If 100% of characters are hex-valid, it's likely hex
	return validChars == len(s) && validChars > 0
}