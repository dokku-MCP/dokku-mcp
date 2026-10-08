package dokkuApi

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestErrorDetail(t *testing.T) {
	out := []byte("-----> Cleaning up...\n !     App web does not exist\n\n")
	if got := errorDetail(out); got != "-----> Cleaning up...; App web does not exist" {
		t.Fatalf("errorDetail = %q", got)
	}
	if got := errorDetail(nil); got != "" {
		t.Fatalf("errorDetail(nil) = %q", got)
	}
	long := []byte(strings.Repeat("x", 2000) + "\nfinal line\n")
	got := errorDetail(long)
	if len(got) > maxErrorDetail+3 || !strings.HasSuffix(got, "final line") {
		t.Fatalf("long output not truncated to its tail: %d chars, suffix %q", len(got), got[len(got)-12:])
	}
}

func TestRedactArgs(t *testing.T) {
	got := RedactArgs("config:set", []string{"--encoded", "--no-restart", "web", "SECRET=c2VjcmV0", "EMPTY="})
	want := []string{"--encoded", "--no-restart", "web", "SECRET=***", "EMPTY=***"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("RedactArgs = %v, want %v", got, want)
	}
	if other := RedactArgs("ps:scale", []string{"web", "web=2"}); other[1] != "web=2" {
		t.Fatalf("non-secret commands must not be redacted: %v", other)
	}
}

func TestRedactArgsMasksURLCredentials(t *testing.T) {
	got := RedactArgs("git:sync", []string{"web", "https://bot:ghp_secret@github.com/acme/web.git", "main"})
	if got[1] != "https://bot:***@github.com/acme/web.git" {
		t.Fatalf("RedactArgs = %v", got)
	}
}

func TestErrorDetailDropsSSHClientNoise(t *testing.T) {
	out := []byte("Warning: Permanently added 'dokku.example.com' (ED25519) to the list of known hosts.\n !     App web does not exist\nConnection to dokku.example.com closed.\n")
	if got := errorDetail(out); got != "App web does not exist" {
		t.Fatalf("errorDetail = %q", got)
	}
}

func TestErrorDetailKeepsNestedSSHErrors(t *testing.T) {
	out := []byte("-----> Syncing git.example.com/acme/web.git\nssh: Could not resolve hostname git.example.com: Name or service not known\nfatal: Could not read from remote repository.\n")
	want := "-----> Syncing git.example.com/acme/web.git; ssh: Could not resolve hostname git.example.com: Name or service not known; fatal: Could not read from remote repository."
	if got := errorDetail(out); got != want {
		t.Fatalf("errorDetail = %q", got)
	}
}

func TestIsTransportFailure(t *testing.T) {
	ssh255 := exec.Command("sh", "-c", "exit 255").Run()
	dokkuFailure := exec.Command("sh", "-c", "exit 1").Run()
	notRunnable := exec.Command("/nonexistent/ssh").Run()

	if !isTransportFailure(ssh255) {
		t.Error("exit status 255 is an ssh connection failure")
	}
	if isTransportFailure(dokkuFailure) {
		t.Error("exit status 1 comes from Dokku, not the transport")
	}
	if !isTransportFailure(notRunnable) {
		t.Error("an ssh binary that cannot start is a transport failure")
	}
	if isTransportFailure(context.DeadlineExceeded) {
		t.Error("timeouts are reported as such, not as unreachable")
	}
}
