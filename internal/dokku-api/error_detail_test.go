package dokkuApi

import (
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
