package base

import (
	"errors"
	"fmt"
	"sync"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/filters"
	"github.com/D00Movenok/BounceBack/internal/wrapper"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	ErrInvalidFilter = errors.New("no such filter")
)

func NewBaseProxy(cfg common.ProxyConfig, fs *filters.FilterSet) (*Proxy, error) {
	logger := log.With().
		Str("proxy", cfg.Name).
		Logger()

	for _, f := range cfg.Filters {
		_, ok := fs.Get(f)
		if !ok {
			return nil, fmt.Errorf("can't find filter \"%s\" for proxy \"%s\"", f, cfg.Name)
		}
	}

	return &Proxy{
		ListenAddr: cfg.Listen,
		Name:       cfg.Name,
		Type:       cfg.Type,

		Logger: logger,

		config:  cfg,
		filters: fs,
	}, nil
}

type Proxy struct {
	ListenAddr string
	Name       string
	Type       string

	Closing bool
	WG      sync.WaitGroup
	Logger  zerolog.Logger

	config  common.ProxyConfig
	filters *filters.FilterSet
}

func (p *Proxy) GetConfig() *common.ProxyConfig {
	return &p.config
}

func (p *Proxy) RunFilters(e wrapper.Entity, logger zerolog.Logger) error {
	for _, f := range p.config.Filters {
		filterLogger := logger.With().Str("filter", f).Logger()
		filter, _ := p.filters.Get(f)
		filtered, err := filter.Apply(e)
		if err != nil {
			filterLogger.Error().Err(err).Msg("Filter error")
			continue
		}
		if filtered {
			return fmt.Errorf("filtered with \"%s\"", f)
		}
	}

	return nil
}

func (p *Proxy) String() string {
	return fmt.Sprintf("%s proxy %s (%s)", p.Type, p.Name, p.ListenAddr)
}
