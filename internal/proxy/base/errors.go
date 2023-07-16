package base

import "errors"

var (
	ErrShutdownTimeout = errors.New("proxy shutdown timeout")
)
