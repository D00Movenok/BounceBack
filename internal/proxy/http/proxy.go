package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/database"
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

func NewProxy(
	cfg common.ProxyConfig,
	fs *filters.FilterSet,
	db *database.DB,
) (*Proxy, error) {
	target, err := url.Parse(cfg.Target)
	if err != nil {
		return nil, fmt.Errorf("can't parse target url: %w", err)
	}

	var action *url.URL
	if cfg.FilterSettings.Action == common.ActionProxy ||
		cfg.FilterSettings.Action == common.ActionRedirect {
		action, err = url.Parse(cfg.FilterSettings.URL)
		if err != nil {
			return nil, fmt.Errorf("can't parse action url: %w", err)
		}
	}

	baseProxy, err := base.NewBaseProxy(cfg, fs, db)
	if err != nil {
		return nil, fmt.Errorf("can't create base proxy: %w", err)
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
		baseProxy.Logger.Debug().Msgf(
			"Using default timeout: %s",
			cfg.Timeout,
		)
	}

	p := &Proxy{
		Proxy:     baseProxy,
		TargetURL: target,
		ActionURL: action,

		client: &http.Client{
			Timeout: cfg.Timeout,
			CheckRedirect: func(
				req *http.Request,
				via []*http.Request,
			) error {
				return http.ErrUseLastResponse
			},
		},
	}

	p.server = &http.Server{
		Addr:         p.Config.Listen,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
		IdleTimeout:  cfg.Timeout,
		Handler:      p.getHandler(),
	}

	if p.TLSConfig != nil {
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

func (p *Proxy) proxyRequest(
	url *url.URL,
	w http.ResponseWriter,
	r *http.Request,
	e wrapper.Entity,
	logger zerolog.Logger,
) {
	r.URL.Scheme = url.Scheme
	r.URL.Host = url.Host

	r.RequestURI = ""
	r.Host = ""
	r.Header.Del("Accept-Encoding")

	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		r.Header.Set("X-Forwarded-For", xForwardedFor+","+e.GetIP().String())
	} else {
		r.Header.Set("X-Forwarded-For", e.GetIP().String())
	}

	response, err := p.client.Do(r)
	if err != nil {
		logger.Error().Err(err).Msg("Can't make proxy request")
		handleError(w)
		return
	}
	defer response.Body.Close()

	for k, vals := range response.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(response.StatusCode)

	if _, err = io.Copy(w, response.Body); err != nil {
		logger.Error().Err(err).Msg("Can't copy body")
		handleError(w)
		return
	}
}

func (p *Proxy) processVerdict(
	w http.ResponseWriter,
	r *http.Request,
	e wrapper.Entity,
	logger zerolog.Logger,
) {
	switch p.Config.FilterSettings.Action {
	case common.ActionProxy:
		p.proxyRequest(p.ActionURL, w, r, e, logger)
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
		logger.Warn().Msg(
			"Request was filtered, but action is None or unknown",
		)
		p.proxyRequest(p.TargetURL, w, r, e, logger)
	}
}

func (p *Proxy) createEntity(r *http.Request) (wrapper.Entity, error) {
	var err error
	if r.Body, err = wrapper.WrapHTTPBody(r.Body); err != nil {
		return nil, fmt.Errorf("can't wrap body: %w", err)
	}
	r.Header.Set("Host", r.Host)

	e := &wrapper.HTTPRequest{Request: r}
	return e, nil
}

func (p *Proxy) getHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		e, err := p.createEntity(r)
		if err != nil {
			p.Logger.Error().Err(err).Msg("Can't create entity")
			handleError(w)
			return
		}

		logger := p.Logger.With().
			Stringer("from", e.GetIP()).
			Logger()

		logRequest(r, logger)
		if !p.RunFilters(e, logger) {
			p.processVerdict(w, r, e, logger)
			return
		}

		p.proxyRequest(p.TargetURL, w, r, e, logger)
	}
}

func (p *Proxy) serve() {
	defer p.WG.Done()
	if p.TLSConfig != nil {
		err := p.server.ListenAndServeTLS("", "")
		if err != nil && err != http.ErrServerClosed {
			p.Logger.Fatal().Err(err).Msg("Unexpected server error")
		}
	} else {
		err := p.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			p.Logger.Fatal().Err(err).Msg("Unexpected server error")
		}
	}
}
