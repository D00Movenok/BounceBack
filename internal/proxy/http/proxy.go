package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/database"
	"github.com/D00Movenok/BounceBack/internal/proxy/base"
	"github.com/D00Movenok/BounceBack/internal/rules"
	"github.com/D00Movenok/BounceBack/internal/wrapper"

	"github.com/rs/zerolog"
)

const (
	ProxyType = "http"
)

var (
	AllowedActions = []string{
		common.RejectActionProxy,
		common.RejectActionRedirect,
		common.RejectActionDrop,
		common.RejectActionNone,
	}
)

type Proxy struct {
	*base.Proxy

	TargetURL *url.URL
	ActionURL *url.URL

	server *http.Server
	client *http.Client
}

func NewProxy(
	cfg common.ProxyConfig,
	rs *rules.RuleSet,
	db *database.DB,
) (*Proxy, error) {
	baseProxy, err := base.NewBaseProxy(cfg, rs, db, AllowedActions)
	if err != nil {
		return nil, fmt.Errorf("can't create base proxy: %w", err)
	}

	target, err := url.Parse(cfg.TargetAddr)
	if err != nil {
		return nil, fmt.Errorf("can't parse target url: %w", err)
	}

	var action *url.URL
	if cfg.RuleSettings.RejectAction == common.RejectActionProxy ||
		cfg.RuleSettings.RejectAction == common.RejectActionRedirect {
		action, err = url.Parse(cfg.RuleSettings.RejectURL)
		if err != nil {
			return nil, fmt.Errorf("can't parse action url: %w", err)
		}
	}

	p := &Proxy{
		Proxy:     baseProxy,
		TargetURL: target,
		ActionURL: action,

		client: &http.Client{
			Timeout: baseProxy.Config.Timeout,
			CheckRedirect: func(
				_ *http.Request,
				_ []*http.Request,
			) error {
				return http.ErrUseLastResponse
			},
		},
	}

	p.server = &http.Server{
		Addr:         p.Config.ListenAddr,
		ReadTimeout:  baseProxy.Config.Timeout,
		WriteTimeout: baseProxy.Config.Timeout,
		IdleTimeout:  baseProxy.Config.Timeout,
		Handler:      p.getHandler(),
	}

	if p.TLSConfig != nil {
		p.client.Transport = &http.Transport{
			TLSClientConfig:   p.TLSConfig,
			ForceAttemptHTTP2: true,
		}
		p.server.TLSConfig = p.TLSConfig
	}

	// TODO: Remove next when HTTP2 will support Drop
	// https://github.com/golang/go/issues/34874
	if cfg.RuleSettings.RejectAction == common.RejectActionDrop {
		p.Logger.Warn().Msg("HTTP2 disabled with action \"drop\"")
		p.server.TLSNextProto = make(
			map[string]func(*http.Server, *tls.Conn, http.Handler),
		)
	}

	return p, nil
}

func (p *Proxy) Start() error {
	p.WG.Add(1)
	go p.serve()
	return nil
}

func (p *Proxy) Shutdown(ctx context.Context) error {
	p.Closing = true
	err := p.server.Shutdown(ctx)
	if err != nil {
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
		return base.ErrShutdownTimeout
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
	r.URL.Path = url.Path + r.URL.Path

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

	_, err = io.Copy(w, response.Body)
	if err != nil {
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
	switch p.Config.RuleSettings.RejectAction {
	case common.RejectActionProxy:
		p.proxyRequest(p.ActionURL, w, r, e, logger)
	case common.RejectActionRedirect:
		http.Redirect(w, r, p.ActionURL.String(), http.StatusMovedPermanently)
	case common.RejectActionDrop:
		hj, ok := w.(http.Hijacker)
		if !ok {
			// TODO: Add support for HTTP2 Hijacker
			// https://github.com/golang/go/issues/34874
			logger.Warn().Msg("Response writer does not support http.Hijacker")
			handleError(w)
			return
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			logger.Error().Err(err).Msg("Can't hijack response")
			handleError(w)
			return
		}
		conn.Close() //nolint:gosec // does not matter if error occurs
	default:
		logger.Warn().Msg("Request was filtered, but action is none")
		p.proxyRequest(p.TargetURL, w, r, e, logger)
	}
}

func (p *Proxy) createEntity(r *http.Request) (wrapper.Entity, error) {
	var err error
	r.Body, err = wrapper.WrapHTTPBody(r.Body)

	if err != nil {
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

		logRequest(e, logger)
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
