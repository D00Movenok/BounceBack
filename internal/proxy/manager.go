package proxy

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/proxy/http"

	"github.com/rs/zerolog/log"
)

var (
	ErrNoSuchProxy = errors.New("no such proxy")
)

func NewManager(cfg *common.ProxyConfig) (*Manager, error) {
	proxies := make([]Proxy, 0)
	for _, pCfg := range cfg.Proxies {
		var (
			p   Proxy
			err error
		)
		switch pCfg.Type {
		case "http":
			if p, err = http.NewProxy(pCfg); err != nil {
				log.Fatal().Err(err).Msg("Error creating http proxy")
			}
		default:
			return nil, fmt.Errorf("invalid proxy type: %s", pCfg.Type)
		}
		proxies = append(proxies, p)
	}

	m := &Manager{proxies}
	return m, nil
}

type Manager struct {
	proxies []Proxy
}

func (m *Manager) StartAll() error {
	for i, p := range m.proxies {
		if err := p.Start(); err != nil {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5) //nolint:gomnd
			for j := 0; j < i; j++ {
				if serr := m.proxies[j].Shutdown(ctx); serr != nil {
					log.Error().Err(err).Msgf("Error shutting down %s forcefully", m.proxies[j])
				}
			}
			cancel()
			return fmt.Errorf("starting %s: %w", p, err)
		}
	}
	return nil
}

func (m *Manager) Shutdown(ctx context.Context) error {
	wg := sync.WaitGroup{}
	wg.Add(len(m.proxies))
	errCh := make(chan error)
	for _, p := range m.proxies {
		go func(p Proxy) {
			defer wg.Done()
			if err := p.Shutdown(ctx); err != nil {
				log.Error().Err(err).Msgf("Error shutting down %s", p)
				select {
				case errCh <- err:
				default:
				}
			}
		}(p)
	}
	wg.Wait()
	select {
	case err := <-errCh:
		return fmt.Errorf("error shutting down proxy: %w", err)
	default:
		return nil
	}
}
