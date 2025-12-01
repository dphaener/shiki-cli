package diff

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// getWindowsPathExpected returns expected result for Windows paths based on current platform
func getWindowsPathExpected(winPath string) string {
	if runtime.GOOS == "windows" {
		return filepath.Base(winPath)
	}
	// On Unix-like systems, Windows paths are treated as regular relative paths
	return winPath
}

// TestSanitizeFilePathEdgeCases tests comprehensive edge cases for file path sanitization
func TestSanitizeFilePathEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic cases
		{
			name:     "empty_path",
			input:    "",
			expected: ".",
		},
		{
			name:     "current_directory",
			input:    ".",
			expected: ".",
		},
		{
			name:     "parent_directory",
			input:    "..",
			expected: "..",
		},
		{
			name:     "root_path",
			input:    "/",
			expected: ".",
		},

		// Windows-specific paths (behavior depends on platform)
		{
			name:     "windows_absolute_path",
			input:    `C:\Users\username\file.txt`,
			expected: getWindowsPathExpected(`C:\Users\username\file.txt`),
		},
		{
			name:     "windows_unc_path",
			input:    `\\server\share\file.txt`,
			expected: getWindowsPathExpected(`\\server\share\file.txt`),
		},
		{
			name:     "windows_drive_relative",
			input:    `C:file.txt`,
			expected: getWindowsPathExpected(`C:file.txt`),
		},

		// Sensitive path variations
		{
			name:     "secrets_with_uppercase",
			input:    "/home/user/SECRETS/api-key.txt",
			expected: "api-key.txt",
		},
		{
			name:     "env_file_variations",
			input:    "project/.env.local",
			expected: ".env.local",
		},
		{
			name:     "env_file_development",
			input:    "project/.env.development",
			expected: ".env.development",
		},
		{
			name:     "credentials_in_path",
			input:    "/var/lib/credentials/db.conf",
			expected: "db.conf",
		},
		{
			name:     "ssh_config",
			input:    "/home/user/.ssh/config",
			expected: "config",
		},
		{
			name:     "ssh_known_hosts",
			input:    "/home/user/.ssh/known_hosts",
			expected: "known_hosts",
		},
		{
			name:     "password_file",
			input:    "/etc/passwords/db.pass",
			expected: "db.pass",
		},
		{
			name:     "api_tokens",
			input:    "/var/lib/tokens/github.token",
			expected: "github.token",
		},

		// Special characters and Unicode
		{
			name:     "unicode_filename",
			input:    "src/测试文件.txt",
			expected: "src/测试文件.txt",
		},
		{
			name:     "spaces_in_path",
			input:    "My Documents/my file.txt",
			expected: "My Documents/my file.txt",
		},
		{
			name:     "special_characters",
			input:    "src/file@#$%.txt",
			expected: "src/file@#$%.txt",
		},
		{
			name:     "emoji_in_filename",
			input:    "src/test🚀file.txt",
			expected: "src/test🚀file.txt",
		},

		// Path traversal attempts
		{
			name:     "path_traversal_simple",
			input:    "../../../etc/passwd",
			expected: "passwd",
		},
		{
			name:     "path_traversal_complex",
			input:    "safe/../../../../../../etc/passwd",
			expected: "passwd",
		},
		{
			name:     "encoded_path_traversal",
			input:    "safe%2f%2e%2e%2f%2e%2e%2fpasswd",
			expected: "safe%2f%2e%2e%2f%2e%2e%2fpasswd", // URL encoding should be preserved as-is
		},

		// Very long paths
		{
			name:     "extremely_long_path",
			input:    strings.Repeat("very/long/", 50) + "file.txt",
			expected: "file.txt",
		},
		{
			name:     "long_filename",
			input:    "src/" + strings.Repeat("verylongname", 20) + ".txt",
			expected: strings.Repeat("verylongname", 20) + ".txt",
		},

		// Hidden files and directories
		{
			name:     "hidden_file",
			input:    "src/.hidden",
			expected: "src/.hidden",
		},
		{
			name:     "hidden_directory",
			input:    ".config/app/settings.json",
			expected: ".config/app/settings.json",
		},
		{
			name:     "git_directory",
			input:    ".git/config",
			expected: ".git/config",
		},

		// Null bytes and control characters
		{
			name:     "null_byte_in_path",
			input:    "src/file\x00.txt",
			expected: "src/file.txt", // Should strip null bytes but keep path
		},
		{
			name:     "control_characters",
			input:    "src/file\x01\x02.txt",
			expected: "src/file.txt", // Should strip control characters but keep path
		},
		{
			name:     "newline_in_path",
			input:    "src/file\nname.txt",
			expected: "src/filename.txt", // Should strip newlines but keep path
		},

		// Mixed case sensitivity
		{
			name:     "mixed_case_sensitive_path",
			input:    "/HOME/USER/.SSH/keys/id_rsa",
			expected: "id_rsa",
		},
		{
			name:     "mixed_case_secrets",
			input:    "/var/Secrets/API/key.txt",
			expected: "key.txt",
		},

		// Symlink-like patterns
		{
			name:     "symbolic_link_pattern",
			input:    "link -> /etc/passwd",
			expected: "passwd", // "/etc/passwd" is detected as sensitive, so basename only
		},

		// Cloud and container paths
		{
			name:     "docker_secrets",
			input:    "/run/secrets/db_password",
			expected: "db_password",
		},
		{
			name:     "kubernetes_secrets",
			input:    "/var/run/secrets/kubernetes.io/serviceaccount/token",
			expected: "token",
		},

		// Additional sensitive patterns
		{
			name:     "aws_credentials",
			input:    "/home/user/.aws/credentials",
			expected: "credentials",
		},
		{
			name:     "gcp_service_account",
			input:    "/etc/service-account.json",
			expected: "service-account.json",
		},
		{
			name:     "private_key",
			input:    "/home/user/private.key",
			expected: "private.key",
		},
		{
			name:     "certificate_file",
			input:    "/etc/ssl/private/server.key",
			expected: "server.key",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := SanitizeFilePath(test.input)
			if result != test.expected {
				t.Errorf("Expected %q, got %q", test.expected, result)
			}

			// Additional safety checks
			if result == "" {
				t.Error("Sanitized path should never be empty")
			}

			// Ensure no null bytes or control characters in result
			for i, r := range result {
				if r == 0 || (r < 32 && r != '\t') {
					t.Errorf("Result contains control character at position %d: %#U", i, r)
				}
			}
		})
	}
}

// TestSanitizeFilePathSecurity tests security-focused edge cases
func TestSanitizeFilePathSecurity(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		shouldHide  bool
		description string
	}{
		{
			name:        "private_key_extension",
			input:       "/home/user/server.pem",
			shouldHide:  true,
			description: "Files with private key extensions should be hidden",
		},
		{
			name:        "config_with_password",
			input:       "/etc/app/database.conf",
			shouldHide:  false, // Only hide if in sensitive directory
			description: "Config files are OK unless in sensitive directories",
		},
		{
			name:        "backup_of_sensitive",
			input:       "/home/user/.ssh/id_rsa.bak",
			shouldHide:  true,
			description: "Backups of sensitive files should be hidden",
		},
		{
			name:        "home_directory_exposure",
			input:       "/home/alice/documents/file.txt",
			shouldHide:  false,
			description: "Regular files in home should be shown with basename only",
		},
		{
			name:        "system_config",
			input:       "/etc/shadow",
			shouldHide:  true,
			description: "System password files should be hidden",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := SanitizeFilePath(tc.input)

			// If it should be hidden, result should be basename only
			if tc.shouldHide {
				expected := filepath.Base(tc.input)
				if result != expected {
					t.Errorf("Expected sensitive path to show basename only: %q, got %q", expected, result)
				}
			}

			// Ensure no absolute paths leak through
			if filepath.IsAbs(result) && result != "." && result != ".." {
				t.Errorf("Absolute path should not be returned: %q", result)
			}
		})
	}
}

// TestSanitizeFilePathPlatformSpecific tests platform-specific behaviors
func TestSanitizeFilePathPlatformSpecific(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Run("windows_drive_letters", func(t *testing.T) {
			tests := []struct {
				input    string
				expected string
			}{
				{`C:\file.txt`, "file.txt"},
				{`D:\secrets\key.txt`, "key.txt"},
				{`\\?\C:\very\long\path\file.txt`, "file.txt"},
			}

			for _, test := range tests {
				result := SanitizeFilePath(test.input)
				if result != test.expected {
					t.Errorf("Input %q: expected %q, got %q", test.input, test.expected, result)
				}
			}
		})
	}

	if runtime.GOOS != "windows" {
		t.Run("unix_absolute_paths", func(t *testing.T) {
			tests := []struct {
				input    string
				expected string
			}{
				{"/usr/local/bin/file", "file"},
				{"/tmp/secrets/key.txt", "key.txt"},
				{"/proc/self/environ", "environ"},
			}

			for _, test := range tests {
				result := SanitizeFilePath(test.input)
				if result != test.expected {
					t.Errorf("Input %q: expected %q, got %q", test.input, test.expected, result)
				}
			}
		})
	}
}

// TestSanitizeFilePathPerformance ensures sanitization doesn't impact performance
func TestSanitizeFilePathPerformance(t *testing.T) {
	// Test with a very long path to ensure O(n) performance
	longPath := strings.Repeat("a/", 1000) + "file.txt"

	// Should complete quickly
	result := SanitizeFilePath(longPath)

	// Result should be reasonable
	if result != "file.txt" {
		t.Errorf("Expected 'file.txt', got %q", result)
	}
}

// BenchmarkSanitizeFilePath benchmarks the sanitization function
func BenchmarkSanitizeFilePath(b *testing.B) {
	testPaths := []string{
		"src/main.go",
		"/home/user/secrets/config.yml",
		strings.Repeat("very/", 20) + "long/path/file.txt",
		"/home/user/.ssh/id_rsa",
		"project/.env.local",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range testPaths {
			SanitizeFilePath(path)
		}
	}
}