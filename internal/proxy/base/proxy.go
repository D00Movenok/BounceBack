package base

import (
	"errors"
	"fmt"
	"sync"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/database"
	"github.com/D00Movenok/BounceBack/internal/filters"
	"github.com/D00Movenok/BounceBack/internal/wrapper"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	ErrInvalidFilter = errors.New("no such filter")
)

func NewBaseProxy(
	cfg common.ProxyConfig,
	fs *filters.FilterSet,
	db *database.DB,
) (*Proxy, error) {
	logger := log.With().
		Str("proxy", cfg.Name).
		Logger()

	for _, f := range cfg.Filters {
		_, ok := fs.Get(f)
		if !ok {
			return nil, fmt.Errorf(
				"can't find filter \"%s\" for proxy \"%s\"",
				f,
				cfg.Name,
			)
		}
	}

	return &Proxy{
		ListenAddr: cfg.Listen,
		TargetAddr: cfg.Target,
		Name:       cfg.Name,
		Type:       cfg.Type,

		Logger: logger,

		db:      db,
		config:  cfg,
		filters: fs,
	}, nil
}

type Proxy struct {
	ListenAddr string
	TargetAddr string
	Name       string
	Type       string

	Closing bool
	WG      sync.WaitGroup
	Logger  zerolog.Logger

	db      *database.DB
	config  common.ProxyConfig
	filters *filters.FilterSet
}

func (p *Proxy) GetConfig() *common.ProxyConfig {
	return &p.config
}

func (p *Proxy) GetFullInfoLogger() *zerolog.Logger {
	logger := p.Logger.With().
		Str("listen", p.ListenAddr).
		Str("target", p.TargetAddr).
		Str("type", p.Type).
		Logger()
	return &logger
}

// Return true if entity passed all checks and false if filtered.
func (p *Proxy) RunFilters(e wrapper.Entity, logger zerolog.Logger) bool {
	ip := e.GetIP().String()

	v, err := p.db.GetVerdict(ip)
	if err != nil {
		v = &database.Verdict{}
		logger.Error().Err(err).Msg("Can't get cached verdict")
	}
	switch {
	case p.config.FilterSettings.NoRejectThreshold > 0 &&
		v.Accepts >= p.config.FilterSettings.NoRejectThreshold:
	case p.config.FilterSettings.RejectThreshold > 0 &&
		v.Rejects >= p.config.FilterSettings.RejectThreshold:
		logger.Warn().Msg("Rejected permanently")
		return false
	default:
	}

	var filtered bool
	for _, f := range p.config.Filters {
		filterLogger := logger.With().Str("filter", f).Logger()
		filter, _ := p.filters.Get(f)
		filtered, err = filter.Apply(e, filterLogger)
		if err != nil {
			filterLogger.Error().Err(err).Msg("Filter error")
			continue
		}
		if filtered {
			filterLogger.Warn().Msg("Filtered")
			err = p.db.IncRejects(ip)
			if err != nil {
				logger.Error().Err(err).Msg("Can't increase rejects")
			}
			return false
		}
	}

	err = p.db.IncAccepts(ip)
	if err != nil {
		logger.Error().Err(err).Msg("Can't increase accepts")
	}

	return true
}

func (p *Proxy) String() string {
	return fmt.Sprintf("%s proxy \"%s\" (%s->%s)",
		p.Type, p.Name, p.ListenAddr, p.TargetAddr)
}
