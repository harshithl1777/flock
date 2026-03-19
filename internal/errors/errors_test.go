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

func TestNewf_FormatsMessage(t *testing.T) {
	err := Newf("parse request", "invalid http method: %s", "TRACE")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if got := err.Error(); got != "parse request: invalid http method: TRACE" {
		t.Fatalf("unexpected formatted error string: %q", got)
	}
}

func TestNewf_ReturnsOpError(t *testing.T) {
	err := Newf("parse request", "invalid content length: %d", 12)

	opErr, ok := err.(*OpError)
	if !ok {
		t.Fatalf("expected *OpError, got %T", err)
	}

	if opErr.Op != "parse request" {
		t.Fatalf("unexpected op: %q", opErr.Op)
	}

	if got := opErr.Unwrap(); got == nil || got.Error() != "invalid content length: 12" {
		t.Fatalf("unexpected wrapped error: %v", got)
	}
}
