package errors

import (
	"errors"
	"fmt"

	"go.uber.org/zap/zapcore"
)

type OpError struct {
	Kind ErrorKind
	Op   string
	Err  error
}

// Error returns the operation name and wrapped error message.
//
// If the receiver or wrapped error is nil, it returns the available message only.
func (e *OpError) Error() string {
	if e == nil {
		return "<nil>"
	}

	if e.Err == nil {
		return e.Op
	}
	return e.Err.Error()
}

// New constructs an OpError for the given operation and message.
//
// The message is wrapped as a concrete error value so it participates in the
// same error-handling flow as other wrapped errors.
func New(kind ErrorKind, op string, msg string) *OpError {
	return &OpError{
		Kind: kind,
		Op:   op,
		Err:  errors.New(msg),
	}
}

// Newf constructs an OpError for the given operation and formatted message.
func Newf(kind ErrorKind, op string, format string, args ...any) *OpError {
	return &OpError{
		Kind: kind,
		Op:   op,
		Err:  fmt.Errorf(format, args...),
	}
}

// MarshalLogObject encodes an OpError as structured fields for zap logs.
func (e *OpError) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if e == nil {
		enc.AddString("error", "<nil>")
		return nil
	}

	enc.AddString("op", e.Op)
	enc.AddString("kind", e.Kind.String())
	if e.Err != nil {
		enc.AddString("error", e.Error())
	}

	return nil
}

// Unwrap returns the underlying error wrapped by the OpError.
//
// A nil receiver returns nil.
func (e *OpError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Err
}

// Wrap annotates err with the provided operation.
//
// It returns nil when err is nil and avoids double-wrapping when err is already
// an OpError for the same operation.
func Wrap(kind ErrorKind, op string, err error) *OpError {
	if err == nil {
		return nil
	}

	return &OpError{
		Kind: kind,
		Op:   op,
		Err:  err,
	}
}

// Chain adds a new operation to an existing OpError while preserving its Kind.
func Chain(op string, err error) *OpError {
	if err == nil {
		return nil
	}

	var opErr *OpError
	if !errors.As(err, &opErr) || opErr == nil {
		return &OpError{
			Op:   op,
			Kind: InternalServerErrorKind,
			Err:  err,
		}
	}

	return &OpError{
		Op:   op,
		Kind: opErr.Kind,
		Err:  err,
	}
}
