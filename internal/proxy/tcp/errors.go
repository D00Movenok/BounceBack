package tcp

import (
	"errors"
	"fmt"
)

var (
	ErrDropped = errors.New("connection dropped")
)

type UnknownShemeError struct {
	scheme string
}

func (e UnknownShemeError) Error() string {
	return fmt.Sprintf("unknown scheme \"%s://\"", e.scheme)
}

type InvalidSchemeAddrPortError struct {
	url string
}

func (e InvalidSchemeAddrPortError) Error() string {
	return fmt.Sprintf("invalid SchemeAddrPort: %s", e.url)
}
