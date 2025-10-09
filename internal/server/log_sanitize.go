package server

import (
	"regexp"
	"strings"
)

// SanitizeLogLines performs minimal redaction on log lines for safe exposure
func SanitizeLogLines(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}
	out := make([]string, len(lines))
	credentialPatterns := []struct {
		regex       *regexp.Regexp
		replacement string
	}{
		{regex: regexp.MustCompile(`(?i)password=[^\s]+`), replacement: "password=[redacted]"},
		{regex: regexp.MustCompile(`(?i)api_key=[^\s]+`), replacement: "api_key=[redacted]"},
		{regex: regexp.MustCompile(`(?i)secret=[^\s]+`), replacement: "secret=[redacted]"},
		{regex: regexp.MustCompile(`(?i)token=[^\s]+`), replacement: "token=[redacted]"},
		{regex: regexp.MustCompile(`(?i)key=[^\s]+`), replacement: "key=[redacted]"},
		{regex: regexp.MustCompile(`(?i)access_key=[^\s]+`), replacement: "access_key=[redacted]"},
		{regex: regexp.MustCompile(`(?i)refresh_token=[^\s]+`), replacement: "refresh_token=[redacted]"},
		{regex: regexp.MustCompile(`(?i)authorization:\s*bearer\s+[a-z0-9\-._~+/=]+`), replacement: "authorization: Bearer [redacted]"},
		{regex: regexp.MustCompile(`ssh-rsa\s+[a-z0-9+/=]+`), replacement: "ssh-rsa [redacted]"},
		{regex: regexp.MustCompile(`(?i)-----BEGIN( RSA)? PRIVATE KEY-----[\s\S]+?-----END( RSA)? PRIVATE KEY-----`), replacement: "[redacted private key]"},
		{regex: regexp.MustCompile(`(?i)https?://[^:@\s]+:[^@\s]+@`), replacement: "http://[redacted]:[redacted]@"},
		{regex: regexp.MustCompile(`(?i)user(name)?[:=]\s*[^\s]+`), replacement: "username=[redacted]"},
		{regex: regexp.MustCompile(`(?i)email=\S+`), replacement: "email=[redacted]"},
		{regex: regexp.MustCompile(`(?i)(aws_|gcp_|azure_)?(access|secret|session)_key[^\s]*=\S+`), replacement: "$1$2_key=[redacted]"},
		{regex: regexp.MustCompile(`(?i)(client\s+id|client\s+secret)[:=]\s*[^\s]+`), replacement: "$1=[redacted]"},
		{regex: regexp.MustCompile(`(?i)(env|environment)\s+variable\s+[A-Z0-9_]+=[^\s]+`), replacement: "environment variable [redacted]"},
		{regex: regexp.MustCompile(`(?i)(password|secret|token)\s*"[^"]+"`), replacement: "$1\"[redacted]\""},
		{regex: regexp.MustCompile(`(?i)(password|secret|token)\s*'[^']+'`), replacement: "$1'[redacted]'"},
	}

	for i, l := range lines {
		l = strings.ReplaceAll(l, " .ssh/", " [redacted]/")
		l = strings.ReplaceAll(l, " dokku@", " user@")
		for _, pattern := range credentialPatterns {
			l = pattern.regex.ReplaceAllString(l, pattern.replacement)
		}
		out[i] = l
	}
	return out
}
