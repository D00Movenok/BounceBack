package base

import (
	"errors"
	"fmt"
	"sync"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/wrapper"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	ErrInvalidFilter = errors.New("no such filter")
)

func NewBaseProxy(cfg common.ProxieConfig) (*Proxy, error) {
	logger := log.With().
		Str("proxy", cfg.Name).
		Logger()

	return &Proxy{
		ListenAddr: cfg.Listen,
		Name:       cfg.Name,
		Type:       cfg.Type,

		Logger: logger,

		proxieConfig: cfg,
	}, nil
}

type Proxy struct {
	ListenAddr string
	Name       string
	Type       string

	Closing bool
	WG      sync.WaitGroup
	Logger  zerolog.Logger

	proxieConfig common.ProxieConfig
}

func (p *Proxy) String() string {
	return fmt.Sprintf("%s proxy %s (%s)", p.Type, p.Name, p.ListenAddr)
}

func (p *Proxy) RunFilters(_ wrapper.Entity) error {
	// process filters
	return nil
}
