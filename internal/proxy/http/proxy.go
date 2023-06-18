package http

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/filters"
	"github.com/D00Movenok/BounceBack/internal/proxy/base"
	"github.com/D00Movenok/BounceBack/internal/wrapper"

	"github.com/rs/zerolog"
)

const (
	ProxyType = "http"

	defaultTimeout = time.Second * 10
)

var (
	ErrShutdownTimeout = errors.New("proxy shutdown timeout")
)

func NewProxy(cfg common.ProxyConfig, fs *filters.FilterSet) (*Proxy, error) {
	target, err := url.Parse(cfg.Target)
	if err != nil {
		return nil, fmt.Errorf("can't parse target url: %w", err)
	}

	var action *url.URL
	if cfg.OnTrigger.Action == common.ActionProxy || cfg.OnTrigger.Action == common.ActionRedirect {
		action, err = url.Parse(cfg.OnTrigger.URL)
		if err != nil {
			return nil, fmt.Errorf("can't parse action url: %w", err)
		}
	}

	baseProxy, err := base.NewBaseProxy(cfg, fs)
	if err != nil {
		return nil, err
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
		baseProxy.Logger.Debug().Msgf("Using default timeout: %s", cfg.Timeout)
	}

	p := &Proxy{
		Proxy:     baseProxy,
		TargetURL: target,
		ActionURL: action,

		client: &http.Client{
			Timeout: cfg.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}

	p.server = &http.Server{
		Addr:         p.ListenAddr,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
		IdleTimeout:  cfg.Timeout,
		Handler:      p.getHandler(),
	}

	if cfg.TLS != nil {
		var cert tls.Certificate
		cert, err = tls.LoadX509KeyPair(cfg.TLS.Cert, cfg.TLS.Key)
		if err != nil {
			return nil, fmt.Errorf("can't load tls config: %w", err)
		}
		// #nosec G402
		p.TLSConfig = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
		}
		p.client.Transport = &http.Transport{
			TLSClientConfig:   p.TLSConfig,
			ForceAttemptHTTP2: true,
		}
		p.server.TLSConfig = p.TLSConfig
	}

	return p, nil
}

type Proxy struct {
	*base.Proxy

	TargetURL *url.URL
	ActionURL *url.URL
	TLSConfig *tls.Config

	server *http.Server
	client *http.Client
}

func (p *Proxy) Start() error {
	p.WG.Add(1)
	go p.serve()
	return nil
}

func (p *Proxy) Shutdown(ctx context.Context) error {
	p.Closing = true
	if err := p.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("can't shutdown server: %w", err)
	}
	p.client.CloseIdleConnections()

	done := make(chan any, 1)
	go func() {
		p.WG.Wait()
		done <- nil
	}()

	select {
	case <-ctx.Done():
		return ErrShutdownTimeout
	case <-done:
		break
	}
	return nil
}

func (p *Proxy) proxyRequest(url *url.URL, w http.ResponseWriter, r *http.Request, logger zerolog.Logger) {
	r.URL.Scheme = url.Scheme
	r.URL.Host = url.Host

	r.RequestURI = ""
	r.Host = ""
	r.Header.Del("Accept-Encoding")

	response, err := p.client.Do(r)
	if err != nil {
		logger.Error().Err(err).Msg("Error making proxy request")
		handleError(w)
		return
	}
	defer response.Body.Close()

	for k, vals := range r.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(response.StatusCode)

	if _, err = io.Copy(w, response.Body); err != nil {
		logger.Error().Err(err).Msg("Error copying body")
		handleError(w)
		return
	}
}

func (p *Proxy) processVerdict(w http.ResponseWriter, r *http.Request, logger zerolog.Logger) {
	cfg := p.GetConfig()
	switch cfg.OnTrigger.Action {
	case common.ActionProxy:
		p.proxyRequest(p.ActionURL, w, r, logger)
	case common.ActionRedirect:
		http.Redirect(w, r, p.ActionURL.String(), http.StatusMovedPermanently)
	case common.ActionDrop:
		hj, _ := w.(http.Hijacker)
		conn, _, err := hj.Hijack()
		if err != nil {
			logger.Error().Err(err).Msg("Can't hijack response")
			handleError(w)
			return
		}
		_ = conn.Close()
	default:
		logger.Warn().Msg("Request was filtered, but action is None")
		p.proxyRequest(p.TargetURL, w, r, logger)
	}
}

func (p *Proxy) processRequest(r *http.Request, logger zerolog.Logger) bool {
	var err error
	if r.Body, err = wrapper.WrapHTTPBody(r.Body); err != nil {
		logger.Error().Err(err).Msg("Error wrapping body")
		return false
	}
	r.Header.Set("Host", r.Host)

	reqEntity := &wrapper.HTTPRequest{Request: r}
	if err = p.RunFilters(reqEntity, logger); err != nil {
		logger.Error().Err(err).Str("action", p.GetConfig().OnTrigger.Action).Msg("Filtered")
		return false
	}

	return true
}

func (p *Proxy) getHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := p.Logger.With().
			Str("from", r.RemoteAddr).
			Logger()

		logRequest(r, logger)
		if ok := p.processRequest(r, logger); !ok {
			p.processVerdict(w, r, logger)
			return
		}

		p.proxyRequest(p.TargetURL, w, r, logger)
	}
}

func (p *Proxy) serve() {
	defer p.WG.Done()
	if p.TLSConfig != nil {
		if err := p.server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			p.Logger.Fatal().Err(err).Msg("Error in server")
		}
	} else {
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			p.Logger.Fatal().Err(err).Msg("Error in server")
		}
	}
}
