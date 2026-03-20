package errors

type ErrorKind int

const (
	ErrMalformedRequestLine ErrorKind = iota
	ErrMalformedHeader
	ErrMissingHost
	ErrUnsupportedVersion
	ErrInvalidContentLength
	ErrConflictingFraming
	ErrUnsupportedTransferEncoding
	ErrIncompleteBody
	ErrHeadersTooLarge
	ErrBodyTooLarge
)

func (k ErrorKind) String() string {
	return ""
}
