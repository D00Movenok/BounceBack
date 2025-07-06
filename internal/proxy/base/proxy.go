package base

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/database"
	"github.com/D00Movenok/BounceBack/internal/rules"
	"github.com/D00Movenok/BounceBack/internal/wrapper"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	defaultTimeout = time.Second * 10
)

type Proxy struct {
	Config    common.ProxyConfig
	TLSConfig *tls.Config

	Closing bool
	WG      sync.WaitGroup
	Logger  zerolog.Logger

	db    *database.DB
	rules *rules.RuleSet
}

func NewBaseProxy(
	cfg common.ProxyConfig,
	rs *rules.RuleSet,
	db *database.DB,
	actions []string,
) (*Proxy, error) {
	logger := log.With().
		Str("proxy", cfg.Name).
		Logger()

	err := verifyAction(cfg.RuleSettings.RejectAction, actions)
	if err != nil {
		return nil, err
	}

	filterActions := []string{
		common.FilterActionAccept,
		common.FilterActionReject,
	}
	for _, f := range cfg.Filters {
		_, ok := rs.Get(f.Rule)
		if !ok {
			return nil, fmt.Errorf(
				"can't find rule \"%s\" for proxy \"%s\"",
				f,
				cfg.Name,
			)
		}
		if !slices.Contains(filterActions, f.Action) {
			return nil, fmt.Errorf(
				"unknown filter action: %s",
				f.Action,
			)
		}
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
		logger.Debug().Msgf(
			"Using default timeout: %s",
			cfg.Timeout,
		)
	}

	base := &Proxy{
		Config: cfg,

		Closing: false,
		Logger:  logger,

		db:    db,
		rules: rs,
	}

	if len(cfg.TLS) > 0 {
		var (
			certs []tls.Certificate
			cert  tls.Certificate
			leaf  *x509.Certificate
		)

		for _, t := range cfg.TLS {
			cert, err = tls.LoadX509KeyPair(t.Cert, t.Key)
			if err != nil {
				return nil, fmt.Errorf("can't load tls certificate: %w", err)
			}

			leaf, err = x509.ParseCertificate(cert.Certificate[0])
			if err != nil {
				return nil, fmt.Errorf("can't parse x509 certificate: %w", err)
			}
			cert.Leaf = leaf

			if len(t.Domains) > 0 {
				cert.Leaf.DNSNames = t.Domains
			}

			logger.Debug().
				Str("cert", t.Cert).
				Str("key", t.Key).
				Any("domains", cert.Leaf.DNSNames).
				Time("valid", cert.Leaf.NotAfter).
				Bool("default", len(certs) == 0).
				Msg("Loaded TLS certificate")

			certs = append(certs, cert)
		}

		//nolint: gosec // ignore tls min version
		base.TLSConfig = &tls.Config{
			Certificates:       certs,
			InsecureSkipVerify: true, // for selfsigned tls client certs
		}
	}

	return base, nil
}

func (p *Proxy) GetLogger() *zerolog.Logger {
	logger := p.Logger.With().
		Str("listen", p.Config.ListenAddr).
		Str("target", p.Config.TargetAddr).
		Str("type", p.Config.Type).
		Logger()
	return &logger
}

// RunFilters return true if entity passed all checks and false if filtered.
func (p *Proxy) RunFilters(e wrapper.Entity, logger zerolog.Logger) bool {
	ip := e.GetIP().String()

	if p.isRejectedByThreshold(ip, logger) {
		return false
	}

	mg := p.prepareRules(e, logger)

	// TODO: cache filters for equal entities for optimization.
	for i, f := range p.Config.Filters {
		mg[i].Lock()
		defer mg[i].Unlock()

		ruleLogger := logger.With().Str("rule", f.Rule).Logger()
		rule, _ := p.rules.Get(f.Rule)

		ruleLogger.Trace().Msg("Applying rule")
		fired, err := rule.Apply(e, ruleLogger)
		if err != nil {
			ruleLogger.Error().Err(err).Msg("Rule error, skipping...")
			continue
		}

		if !fired {
			continue
		}

		ruleLogger.Warn().Str("action", f.Action).Msg("Running action")
		if f.Action == common.FilterActionReject {
			err = p.db.IncRejects(ip)
			if err != nil {
				logger.Error().Err(err).Msg("Can't increase rejects")
			}
			return false
		}

		// accept action
		break
	}

	logger.Debug().Msg("Accepted")
	err := p.db.IncAccepts(ip)
	if err != nil {
		logger.Error().Err(err).Msg("Can't increase accepts")
	}

	return true
}

// check NoRejectThreshold and RejectThreshold.
// return true if rejected by RejectThreshold, otherwise false.
func (p *Proxy) isRejectedByThreshold(ip string, logger zerolog.Logger) bool {
	v, err := p.db.GetVerdict(ip)
	if err != nil {
		v = &database.Verdict{}
		logger.Error().Err(err).Msg("Can't get cached verdict")
	}
	switch {
	case p.Config.RuleSettings.NoRejectThreshold > 0 &&
		v.Accepts >= p.Config.RuleSettings.NoRejectThreshold:
		logger.Debug().Msg("Non-rejected permanently")
	case p.Config.RuleSettings.RejectThreshold > 0 &&
		v.Rejects >= p.Config.RuleSettings.RejectThreshold:
		logger.Warn().Msg("Rejected permanently")
		return true
	default:
	}

	return false
}

// run all requests (e.g. DNS PTR, GEO) concurently for optimisation.
func (p *Proxy) prepareRules(
	e wrapper.Entity,
	logger zerolog.Logger,
) []sync.Mutex {
	mg := make([]sync.Mutex, len(p.Config.Filters))
	for i, f := range p.Config.Filters {
		mg[i].Lock()
		go func(index int, ff common.Filter) {
			defer mg[index].Unlock()

			ruleLogger := logger.With().Str("rule", ff.Rule).Logger()
			rule, _ := p.rules.Get(ff.Rule)
			err := rule.Prepare(e, ruleLogger)
			if err != nil {
				ruleLogger.Error().Err(err).Msg("Prepare error, skipping...")
			}
		}(i, f)
	}
	return mg
}

func (p *Proxy) String() string {
	return fmt.Sprintf("%s proxy \"%s\" (%s->%s)",
		p.Config.Type, p.Config.Name, p.Config.ListenAddr, p.Config.TargetAddr)
}
