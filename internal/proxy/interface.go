package proxy

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
)

type Proxy interface {
	Start() error
	Shutdown(context.Context) error
	GetFullInfoLogger() *zerolog.Logger

	fmt.Stringer
}
