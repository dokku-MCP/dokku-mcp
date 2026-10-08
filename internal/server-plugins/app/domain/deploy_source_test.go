package app

import "testing"

func TestDeploySourceRepoURL(t *testing.T) {
	cases := []struct {
		source DeploySource
		want   string
	}{
		{DeploySource{Type: "git-sync", Metadata: "https://github.com/acme/web.git#0a1b2c"}, "https://github.com/acme/web.git"},
		{DeploySource{Type: "git-sync", Metadata: "git@github.com:acme/web.git"}, "git@github.com:acme/web.git"},
		{DeploySource{Type: "git-push", Metadata: "0a1b2c"}, ""},
		{DeploySource{}, ""},
	}
	for _, c := range cases {
		if got := c.source.RepoURL(); got != c.want {
			t.Errorf("%+v.RepoURL() = %q, want %q", c.source, got, c.want)
		}
	}
}
