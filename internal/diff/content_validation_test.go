package diff

import (
	"strings"
	"testing"
)

// TestValidateFileContentComprehensive tests enhanced file content validation
func TestValidateFileContentComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		// Safe content
		{
			name:     "normal_code",
			content:  "func main() {\n    fmt.Println(\"Hello World\")\n}",
			expected: true,
		},
		{
			name:     "safe_config",
			content:  "host=localhost\nport=3000\ndatabase=myapp\ndebug=true",
			expected: true,
		},
		{
			name:     "documentation",
			content:  "# Configuration Guide\n\nThis file contains application settings.",
			expected: true,
		},

		// Basic credential patterns
		{
			name:     "password_equals",
			content:  "database_config:\n  password=secret123",
			expected: false,
		},
		{
			name:     "api_key_colon",
			content:  "config:\n  api_key: abc123def456",
			expected: false,
		},
		{
			name:     "token_assignment",
			content:  "AUTH_TOKEN=bearer_token_value",
			expected: false,
		},

		// Cloud provider credentials
		{
			name:     "aws_credentials",
			content:  "export AWS_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE",
			expected: false,
		},
		{
			name:     "azure_secret",
			content:  "AZURE_CLIENT_SECRET=very-secret-value",
			expected: false,
		},
		{
			name:     "gcp_service_account",
			content:  "GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json",
			expected: false,
		},

		// Database credentials
		{
			name:     "mysql_password",
			content:  "mysql_password=supersecret",
			expected: false,
		},
		{
			name:     "connection_string",
			content:  "connection_string=postgresql://user:pass@localhost/db",
			expected: false,
		},
		{
			name:     "jdbc_url",
			content:  "db.url=jdbc:mysql://localhost:3306/mydb?user=admin&password=secret",
			expected: false,
		},

		// SSH keys and certificates
		{
			name:     "rsa_private_key",
			content:  "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...",
			expected: false,
		},
		{
			name:     "openssh_private_key",
			content:  "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEA...",
			expected: false,
		},
		{
			name:     "ssh_public_key",
			content:  "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB... user@host",
			expected: false,
		},
		{
			name:     "certificate",
			content:  "-----BEGIN CERTIFICATE-----\nMIIDXTCCAkWgAwIBAgIJA...",
			expected: false,
		},

		// Base64 encoded secrets
		{
			name:     "base64_secret",
			content:  "secret_key = \"dGhpc2lzYXZlcnlsb25nc2VjcmV0a2V5dGhhdGxvb2tzc3VzcGljaW91cw==\"",
			expected: false,
		},
		{
			name:     "base64_multiline",
			content:  "private_key = \"dGhpc2lzYXZlcnlsb25nc2VjcmV0a2V5dGhhdGlzdXNlZGZvcnRlc3Rpbmdwcml2YXRla2V5cw==\"",
			expected: false,
		},
		{
			name:     "short_base64_safe",
			content:  "data = \"aGVsbG8=\"", // "hello" in base64, too short to be flagged
			expected: true,
		},

		// Hexadecimal secrets
		{
			name:     "long_hex_secret",
			content:  "api_secret = \"a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456\"",
			expected: false,
		},
		{
			name:     "hex_with_prefix",
			content:  "encryption_key = \"0xdeadbeefcafebabe1234567890abcdef1234567890abcdef1234567890abcdef\"",
			expected: false,
		},
		{
			name:     "short_hex_safe",
			content:  "color = \"#ff0000\"", // CSS color, too short to be flagged
			expected: true,
		},

		// Various credential formats
		{
			name:     "bearer_token",
			content:  "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
			expected: false,
		},
		{
			name:     "webhook_secret",
			content:  "webhook_secret=abcdef1234567890abcdef1234567890",
			expected: false,
		},
		{
			name:     "session_secret",
			content:  "session_secret: \"very-long-session-secret-key-here\"",
			expected: false,
		},

		// Edge cases and false positives to avoid
		{
			name:     "password_in_comment",
			content:  "# Don't put your password here\nhost=localhost",
			expected: true,
		},
		{
			name:     "username_without_password",
			content:  "username: admin\nhost: localhost",
			expected: false, // "username:" is flagged as potentially sensitive
		},
		{
			name:     "function_name_with_password",
			content:  "function validatePassword() {\n  return true;\n}",
			expected: true, // Function names should be safe
		},
		{
			name:     "log_message",
			content:  "console.log('User authentication successful');",
			expected: true,
		},

		// Mixed content
		{
			name:     "config_with_secret",
			content:  "host=localhost\nport=3000\napi_key=secret123\ndebug=true",
			expected: false,
		},
		{
			name:     "mostly_safe_with_one_secret",
			content:  strings.Repeat("safe_setting=value\n", 10) + "password=secret\n",
			expected: false,
		},

		// Real-world examples (sanitized)
		{
			name:     "docker_compose",
			content:  "version: '3'\nservices:\n  db:\n    environment:\n      POSTGRES_PASSWORD: secret123",
			expected: false,
		},
		{
			name:     "kubernetes_secret",
			content:  "apiVersion: v1\nkind: Secret\ndata:\n  password: \"c2VjcmV0\"", // "secret" in base64
			expected: false,
		},
		{
			name:     "terraform_vars",
			content:  "variable \"db_password\" {\n  default = \"changeme\"\n}",
			expected: true, // This doesn't contain explicit patterns, just a variable name
		},

		// Empty and malformed content
		{
			name:     "empty_content",
			content:  "",
			expected: true,
		},
		{
			name:     "whitespace_only",
			content:  "   \n  \t  \n   ",
			expected: true,
		},
		{
			name:     "malformed_assignment",
			content:  "password===invalid",
			expected: false, // Still contains "password="
		},

		// Language-specific patterns
		{
			name:     "python_env_var",
			content:  "import os\nAPI_KEY = os.getenv('API_KEY', 'default_secret_value')",
			expected: true, // No explicit sensitive patterns, just variable names
		},
		{
			name:     "javascript_config",
			content:  "const config = {\n  apiKey: process.env.API_KEY || 'fallback_key'\n};",
			expected: true, // No explicit sensitive patterns, just variable names
		},
		{
			name:     "java_properties",
			content:  "# Application configuration\napp.database.password=secret\napp.debug=true",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := ValidateFileContent(test.content)
			if result != test.expected {
				t.Errorf("Expected %t, got %t for content: %q", test.expected, result, test.content[:min(50, len(test.content))])
			}
		})
	}
}

// TestBase64Detection tests the base64 detection functionality
func TestBase64Detection(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "obvious_base64_secret",
			content:  "secret=\"dGhpc2lzYXZlcnlsb25nc2VjcmV0a2V5dGhhdGxvb2tzc3VzcGljaW91cw==\"",
			expected: true,
		},
		{
			name:     "short_base64",
			content:  "data=\"aGVsbG8=\"", // "hello", too short
			expected: false,
		},
		{
			name:     "not_base64",
			content:  "message=\"this is not base64 encoded\"",
			expected: false,
		},
		{
			name:     "base64_in_comment",
			content:  "# Example: dGhpc2lzYXZlcnlsb25nc2VjcmV0a2V5dGhhdGxvb2tzc3VzcGljaW91cw==",
			expected: false, // Comments are skipped
		},
		{
			name:     "jwt_token",
			content:  "token=\"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c\"",
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := containsSuspiciousBase64(test.content)
			if result != test.expected {
				t.Errorf("Expected %t, got %t for base64 detection", test.expected, result)
			}
		})
	}
}

// TestHexDetection tests the hexadecimal detection functionality
func TestHexDetection(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "long_hex_secret",
			content:  "key=\"deadbeefcafebabe1234567890abcdef1234567890abcdef1234567890abcdef\"",
			expected: true,
		},
		{
			name:     "hex_with_prefix",
			content:  "secret=\"0xdeadbeefcafebabe1234567890abcdef1234567890abcdef\"",
			expected: true,
		},
		{
			name:     "short_hex",
			content:  "color=\"#ff0000\"",
			expected: false, // Too short
		},
		{
			name:     "not_hex",
			content:  "id=\"not-a-hex-string-at-all\"",
			expected: false,
		},
		{
			name:     "mixed_case_hex",
			content:  "encryption_key=\"DeadBeefCafeBabe1234567890AbCdEf1234567890AbCdEf\"",
			expected: true,
		},
		{
			name:     "hex_in_comment",
			content:  "# Example: deadbeef1234567890abcdef1234567890abcdef1234567890abcdef",
			expected: false, // Comments are skipped
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := containsSuspiciousHexStrings(test.content)
			if result != test.expected {
				t.Errorf("Expected %t, got %t for hex detection", test.expected, result)
			}
		})
	}
}

// TestBase64Helper tests the base64 validation helper function
func TestBase64Helper(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid_base64",
			input:    "dGhpc2lzYXZlcnlsb25nc2VjcmV0a2V5dGhhdGxvb2tzc3VzcGljaW91cw==",
			expected: true,
		},
		{
			name:     "short_string",
			input:    "abc",
			expected: false,
		},
		{
			name:     "not_base64",
			input:    "this-is-not-base64-@#$%^&*()",
			expected: false,
		},
		{
			name:     "mostly_base64",
			input:    "dGhpc2lzYXZlcnlsb25nc2VjcmV0a2V5!!!",
			expected: false, // Contains invalid chars
		},
		{
			name:     "jwt_token_part",
			input:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := isLikelyBase64(test.input)
			if result != test.expected {
				t.Errorf("Expected %t, got %t for input: %s", test.expected, result, test.input)
			}
		})
	}
}

// TestHexHelper tests the hex validation helper function
func TestHexHelper(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid_hex",
			input:    "deadbeef1234567890abcdef",
			expected: true,
		},
		{
			name:     "hex_with_prefix",
			input:    "0xdeadbeef1234567890abcdef",
			expected: true,
		},
		{
			name:     "uppercase_hex",
			input:    "DEADBEEF1234567890ABCDEF",
			expected: true,
		},
		{
			name:     "mixed_case_hex",
			input:    "DeadBeef1234567890AbCdEf",
			expected: true,
		},
		{
			name:     "short_hex",
			input:    "ff00",
			expected: false, // Too short
		},
		{
			name:     "not_hex",
			input:    "this-is-not-hex-ghijklmnop",
			expected: false,
		},
		{
			name:     "empty_string",
			input:    "",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := isLikelyHex(test.input)
			if result != test.expected {
				t.Errorf("Expected %t, got %t for input: %s", test.expected, result, test.input)
			}
		})
	}
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}