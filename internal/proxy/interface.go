package proxy

import (
	"context"
	"fmt"
)

type Proxy interface {
	Start() error
	Shutdown(context.Context) error

	fmt.Stringer
}
