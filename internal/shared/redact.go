package shared

import "regexp"

// urlCredentials matches the password of scheme://user:password@host. The
// password may contain '@' but not '/' or whitespace, so the match ends at
// the last '@' of the URL authority and never spills into a following URL.
var urlCredentials = regexp.MustCompile(`(://[^:/@\s]*:)[^/\s]*@`)

// RedactURLCredentials masks passwords embedded in URLs, e.g.
// "postgres://u:secret@host/db" becomes "postgres://u:***@host/db".
func RedactURLCredentials(s string) string {
	return urlCredentials.ReplaceAllString(s, "${1}***@")
}
