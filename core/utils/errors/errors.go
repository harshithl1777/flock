package errors

import "fmt"

type OpError struct {
	Op  string
	Err error
}

// Error returns the operation name and wrapped error message.
//
// If the wrapped error is nil, it returns only the operation name.
func (e *OpError) Error() string {
	if e == nil {
		return "<nil>"
	}

	if e.Err == nil {
		return e.Op
	}
	return e.Op + ": " + e.Err.Error()
}

// New constructs an OpError for the given operation and message.
//
// The message is wrapped as a concrete error value so it can participate in
// the same error-handling flow as other wrapped errors.
func New(op string, msg string) error {
	return &OpError{
		Op:  op,
		Err: fmt.Errorf("%s", msg),
	}
}

// Unwrap returns the underlying error wrapped by the OpError.
func (e *OpError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Err
}

// Wrap annotates err with the provided operation.
//
// It returns nil when err is nil, and avoids double-wrapping when err is
// already an OpError for the same operation.
func Wrap(op string, err error) error {
	if err == nil {
		return nil
	}

	if opErr, ok := err.(*OpError); ok && opErr != nil && opErr.Op == op {
		return err
	}
	return &OpError{
		Op:  op,
		Err: err,
	}
}
