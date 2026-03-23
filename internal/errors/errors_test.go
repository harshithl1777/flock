package errors

import (
	"errors"
	"testing"

	"go.uber.org/zap/zapcore"
)

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

	wrapped := Wrap(MalformedRequestLineKind, "read request", err)
	if wrapped == nil {
		t.Fatal("expected wrapped error, got nil")
	}

	if wrapped.Kind != MalformedRequestLineKind {
		t.Fatalf("got kind %v, want %v", wrapped.Kind, MalformedRequestLineKind)
	}

	if got := wrapped.Error(); got != "<nil>" {
		t.Fatalf("unexpected wrapped error string: %q", got)
	}
}

func TestNewf_FormatsMessage(t *testing.T) {
	err := Newf(MalformedRequestLineKind, "parse request", "invalid http method: %s", "TRACE")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if got := err.Error(); got != "invalid http method: TRACE" {
		t.Fatalf("unexpected formatted error string: %q", got)
	}
}

func TestNewf_ReturnsOpError(t *testing.T) {
	err := Newf(InvalidContentLengthKind, "parse request", "invalid content length: %d", 12)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.Op != "parse request" {
		t.Fatalf("unexpected op: %q", err.Op)
	}

	if err.Kind != InvalidContentLengthKind {
		t.Fatalf("unexpected kind: %v", err.Kind)
	}

	if got := err.Unwrap(); got == nil || got.Error() != "invalid content length: 12" {
		t.Fatalf("unexpected wrapped error: %v", got)
	}
}

func TestChain_PreservesKindForOpErrors(t *testing.T) {
	root := New(MalformedHeaderKind, "parse headers", "missing colon")

	chained := Chain("read request", root)

	if chained.Kind != MalformedHeaderKind {
		t.Fatalf("got kind %v, want %v", chained.Kind, MalformedHeaderKind)
	}

	if !errors.Is(chained, root) {
		t.Fatalf("expected chained error to wrap original error")
	}
}

func TestChain_NilErrorReturnsNil(t *testing.T) {
	if got := Chain("read request", nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestOpErrorMarshalLogObject_NilReceiver(t *testing.T) {
	var opErr *OpError
	enc := zapcore.NewMapObjectEncoder()

	if err := opErr.MarshalLogObject(enc); err != nil {
		t.Fatalf("MarshalLogObject returned error: %v", err)
	}

	if got := enc.Fields["error"]; got != "<nil>" {
		t.Fatalf("got error field %v, want <nil>", got)
	}
}
