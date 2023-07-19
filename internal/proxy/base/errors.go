package base

import "errors"

var (
	ErrShutdownTimeout = errors.New("proxy shutdown timeout")
	ErrDropped         = errors.New("connection dropped")
	ErrTLSUnsupported  = errors.New("TLS is unsopported")
)
