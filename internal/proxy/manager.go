package proxy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/database"
	"github.com/D00Movenok/BounceBack/internal/proxy/dns"
	"github.com/D00Movenok/BounceBack/internal/proxy/http"
	"github.com/D00Movenok/BounceBack/internal/proxy/tcp"
	"github.com/D00Movenok/BounceBack/internal/proxy/udp"
	"github.com/D00Movenok/BounceBack/internal/rules"

	"github.com/rs/zerolog/log"
)

type Manager struct {
	proxies []Proxy
}

func NewManager(db *database.DB, cfg *common.Config) (*Manager, error) {
	rs, err := rules.NewRuleSet(db, cfg.Rules, cfg.Globals)
	if err != nil {
		return nil, fmt.Errorf("can't create rules: %w", err)
	}

	proxies := make([]Proxy, len(cfg.Proxies))
	for i, pc := range cfg.Proxies {
		log.Trace().Any("proxy_cfg", pc).Msg("Creating proxy")
		switch pc.Type {
		case http.ProxyType:
			proxies[i], err = http.NewProxy(pc, rs, db)
		case dns.ProxyType:
			proxies[i], err = dns.NewProxy(pc, rs, db)
		case tcp.ProxyType:
			proxies[i], err = tcp.NewProxy(pc, rs, db)
		case udp.ProxyType:
			proxies[i], err = udp.NewProxy(pc, rs, db)
		default:
			return nil, &InvalidProxyTypeError{t: pc.Type}
		}
		if err != nil {
			return nil, fmt.Errorf(
				"can't create proxy \"%s\": %w",
				pc.Name,
				err,
			)
		}
		proxies[i].GetLogger().Debug().Msg("Created new proxy")
	}

	m := &Manager{proxies}
	return m, nil
}

func (m *Manager) StartAll() error {
	for i, p := range m.proxies {
		p.GetLogger().Info().Msg("Starting proxy")
		err := p.Start()
		if err != nil {
			ctx, cancel := context.WithTimeout(
				context.Background(),
				time.Second*5, //nolint:mnd
			)
			defer cancel()
			for _, rp := range m.proxies[:i] {
				serr := rp.Shutdown(ctx)
				if serr != nil {
					log.Error().Err(serr).Msgf(
						"Error shutting down %s forcefully",
						rp,
					)
				}
			}
			return fmt.Errorf("can't start \"%s\": %w", p, err)
		}
	}
	return nil
}

func (m *Manager) Shutdown(ctx context.Context) error {
	wg := sync.WaitGroup{}
	wg.Add(len(m.proxies))
	errCh := make(chan error)
	for _, p := range m.proxies {
		p.GetLogger().Info().Msg("Shutting down proxy")
		go func(p Proxy) {
			defer wg.Done()
			err := p.Shutdown(ctx)
			if err != nil {
				select {
				case errCh <- fmt.Errorf("can't shutdown \"%s\": %w", p, err):
				default:
				}
			}
		}(p)
	}
	wg.Wait()
	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}
