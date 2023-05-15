package proxy

import (
	"context"
	"fmt"
)

type Proxy interface {
	Start() error
	Shutdown(ctx context.Context) error

	fmt.Stringer
}
