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
