package errors

import "fmt"

type OpError struct {
	Op  string
	Err error
}

func (e *OpError) Error() string {
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *OpError) Unwrap() error {
	return e.Err
}

func Wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	return &OpError{
		Op:  op,
		Err: err,
	}
}
