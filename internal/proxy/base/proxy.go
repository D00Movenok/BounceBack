package base

import (
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/database"
	"github.com/D00Movenok/BounceBack/internal/filters"
	"github.com/D00Movenok/BounceBack/internal/wrapper"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

	base := &Proxy{
		Config: cfg,

		Closing: false,
		Logger:  logger,

		db:      db,
		filters: fs,
	}

	if cfg.TLS != nil {
		cert, err := tls.LoadX509KeyPair(cfg.TLS.Cert, cfg.TLS.Key)
		if err != nil {
			return nil, fmt.Errorf("can't load tls config: %w", err)
		}
		// #nosec G402
		base.TLSConfig = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
		}
	}

	return base, nil
}

type Proxy struct {
	Config    common.ProxyConfig
	TLSConfig *tls.Config

	Closing bool
	WG      sync.WaitGroup
	Logger  zerolog.Logger

	db      *database.DB
	filters *filters.FilterSet
}

func (p *Proxy) GetLogger() *zerolog.Logger {
	logger := p.Logger.With().
		Str("listen", p.Config.Listen).
		Str("target", p.Config.Target).
		Str("type", p.Config.Type).
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
	case p.Config.FilterSettings.NoRejectThreshold > 0 &&
		v.Accepts >= p.Config.FilterSettings.NoRejectThreshold:
	case p.Config.FilterSettings.RejectThreshold > 0 &&
		v.Rejects >= p.Config.FilterSettings.RejectThreshold:
		logger.Warn().Msg("Rejected permanently")
		return false
	default:
	}

	// TODO: run all requests (e.g. DNS PTR, GEO) concurrently
	// before filtering for optimization.
	// TODO: cache filters for equal entities for optimization.
	var filtered bool
	for _, f := range p.Config.Filters {
		filterLogger := logger.With().Str("filter", f).Logger()
		filter, _ := p.filters.Get(f)
		filtered, err = filter.Apply(e, filterLogger)
		if err != nil {
			filterLogger.Error().Err(err).Msg("Filter error, skipping...")
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
		p.Config.Type, p.Config.Name, p.Config.Listen, p.Config.Target)
}
