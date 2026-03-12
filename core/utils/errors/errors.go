package errors

import "fmt"

type OpError struct {
	Op  string
	Err error
}

func (e *OpError) Error() string {
	if e.Err == nil {
		return e.Op
	}
	return e.Op + ": " + e.Err.Error()
}

func New(op string, msg string) error {
	return &OpError{
		Op:  op,
		Err: fmt.Errorf("%s", msg),
	}
}

func (e *OpError) Unwrap() error {
	return e.Err
}

func Wrap(op string, err error) error {
	if err == nil {
		return nil
	}

	if opErr, ok := err.(*OpError); ok && opErr.Op == op {
		return err
	}
	return &OpError{
		Op:  op,
		Err: err,
	}
}
