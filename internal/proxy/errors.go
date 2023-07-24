package proxy

import "fmt"

type InvalidProxyTypeError struct {
	t string
}

func (e InvalidProxyTypeError) Error() string {
	return fmt.Sprintf("invalid proxy type: %s", e.t)
}
