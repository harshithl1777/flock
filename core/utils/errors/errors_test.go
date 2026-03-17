package errors

import "testing"

func TestOpErrorError_NilReceiver(t *testing.T) {
	var opErr *OpError

	if got := opErr.Error(); got != "<nil>" {
		t.Fatalf("expected nil receiver string, got %q", got)
	}
}

func TestOpErrorUnwrap_NilReceiver(t *testing.T) {
	var opErr *OpError

	if got := opErr.Unwrap(); got != nil {
		t.Fatalf("expected nil unwrap result, got %v", got)
	}
}

func TestWrap_TypedNilOpErrorDoesNotPanic(t *testing.T) {
	var opErr *OpError
	var err error = opErr

	wrapped := Wrap("read request", err)
	if wrapped == nil {
		t.Fatal("expected wrapped error, got nil")
	}

	if got := wrapped.Error(); got != "read request: <nil>" {
		t.Fatalf("unexpected wrapped error string: %q", got)
	}
}
