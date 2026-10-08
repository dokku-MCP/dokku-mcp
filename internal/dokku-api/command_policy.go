package dokkuApi

import (
	"fmt"
	"strings"
)

// Dokku's SSH forced command expands $SSH_ORIGINAL_COMMAND unquoted, so the
// remote shell word-splits every argument on whitespace and expands glob
// characters. Quoting cannot survive that, so arguments are restricted to a
// conservative character set and free-form values (such as environment
// variable values) must be sent base64-encoded (see config:set --encoded).

// isSafeArgRune reports whether r may appear anywhere in a Dokku argument.
func isSafeArgRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	}
	return strings.ContainsRune("-_.:/=@+,%", r)
}

// isSafeInnerArgRune reports whether r may appear in a Dokku argument except
// as its first character, where the shell would give it a special meaning
// (comment or home-directory expansion).
func isSafeInnerArgRune(r rune) bool {
	return r == '#' || r == '~'
}

// ValidateArg checks that a single argument survives the trip through the
// remote shell unchanged.
func ValidateArg(arg string) error {
	if arg == "" {
		return fmt.Errorf("argument must not be empty")
	}
	for i, r := range arg {
		if isSafeArgRune(r) || (i > 0 && isSafeInnerArgRune(r)) {
			continue
		}
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return fmt.Errorf("argument %q contains whitespace, which Dokku's SSH command would split", arg)
		}
		return fmt.Errorf("argument %q contains unsupported character %q", arg, r)
	}
	return nil
}

// validateCommandName checks that a Dokku command name only contains
// letters, digits, dashes and colons.
func validateCommandName(commandName string) error {
	if commandName == "" {
		return fmt.Errorf("command name cannot be empty")
	}
	for _, r := range commandName {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != ':' {
			return fmt.Errorf("command name contains invalid character %q: %s", r, commandName)
		}
	}
	return nil
}

// MatchesCommandPattern reports whether commandName matches an allow-list
// pattern. A pattern ending in ':' or '*' matches by prefix ("postgres:" or
// "apps:*" match every postgres or apps command); "*" alone matches
// everything; any other pattern must match exactly.
func MatchesCommandPattern(commandName, pattern string) bool {
	switch {
	case pattern == "*":
		return true
	case strings.HasSuffix(pattern, "*"):
		return strings.HasPrefix(commandName, strings.TrimSuffix(pattern, "*"))
	case strings.HasSuffix(pattern, ":"):
		return strings.HasPrefix(commandName, pattern)
	default:
		return commandName == pattern
	}
}

// cacheableSuffixes lists the subcommand suffixes of read-only Dokku
// commands whose output is safe to cache briefly.
var cacheableSuffixes = []string{":report", ":list", ":show", ":exists", ":info", ":get", ":keys"}

// IsCacheableCommand reports whether a command is a read-only query whose
// result may be served from cache. Logs and events are excluded because
// callers expect them to be fresh.
func IsCacheableCommand(commandName string) bool {
	if commandName == "version" {
		return true
	}
	for _, suffix := range cacheableSuffixes {
		if strings.HasSuffix(commandName, suffix) {
			return true
		}
	}
	return false
}

// isReadOnlyCommand reports whether a command never changes server state, so
// executing it does not need to invalidate cached query results.
func isReadOnlyCommand(commandName string) bool {
	if IsCacheableCommand(commandName) {
		return true
	}
	switch commandName {
	case "logs", "events", "logs:failed", "ps:inspect", "config:export":
		return true
	}
	return false
}
