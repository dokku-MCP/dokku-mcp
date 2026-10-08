package shared

import "testing"

func TestRedactURLCredentials(t *testing.T) {
	cases := map[string]string{
		"redis://:pass@host:6379":                   "redis://:***@host:6379",
		"postgres://user:p@ss@host:5432/db":         "postgres://user:***@host:5432/db",
		"https://bot:ghp_token@github.com/acme/web": "https://bot:***@github.com/acme/web",
		"postgres://u:p@h,redis://r:p2@h2":          "postgres://u:***@h,redis://r:***@h2",
		"git@github.com:acme/web.git":               "git@github.com:acme/web.git",
		"postgres://host:5432/db":                   "postgres://host:5432/db",
	}
	for in, want := range cases {
		if got := RedactURLCredentials(in); got != want {
			t.Errorf("RedactURLCredentials(%q) = %q, want %q", in, got, want)
		}
	}
}
