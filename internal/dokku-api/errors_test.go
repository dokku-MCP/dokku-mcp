package dokkuApi

import "testing"

func TestIsNotFoundError(t *testing.T) {
	var err error
	if IsNotFoundError(err) {
		t.Fatalf("nil should not be not-found")
	}

	err = &NotFoundError{Command: "apps:report"}
	if !IsNotFoundError(err) {
		t.Fatalf("expected not-found classification")
	}

	if !IsNotFoundError(ErrAppNotFound) {
		t.Fatalf("sentinel should be classified not-found")
	}
}
