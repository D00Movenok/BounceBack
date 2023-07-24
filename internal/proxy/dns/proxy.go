package dns

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/database"
	"github.com/D00Movenok/BounceBack/internal/filters"
	"github.com/D00Movenok/BounceBack/internal/proxy/base"
	"github.com/D00Movenok/BounceBack/internal/wrapper"
	"github.com/miekg/dns"

	"github.com/rs/zerolog"
)

const (
	ProxyType = "dns"
)

var (
	AllowedActions = []string{
		common.ActionProxy,
		common.ActionDrop,
		common.ActionNone,
	}
)

func NewProxy(
	cfg common.ProxyConfig,
	fs *filters.FilterSet,
	db *database.DB,
) (*Proxy, error) {
	baseProxy, err := base.NewBaseProxy(cfg, fs, db, AllowedActions)
	if err != nil {
		return nil, fmt.Errorf("can't create base proxy: %w", err)
	}

	target, err := netip.ParseAddrPort(cfg.TargetAddr)
	if err != nil {
		return nil, fmt.Errorf("can't parse AddrPort: %w", err)
	}

	var action netip.AddrPort
	if cfg.FilterSettings.Action == common.ActionProxy ||
		cfg.FilterSettings.Action == common.ActionRedirect {
		action, err = netip.ParseAddrPort(cfg.FilterSettings.URL)
		if err != nil {
			return nil, fmt.Errorf("can't parse AddrPort: %w", err)
		}
	}

	p := &Proxy{
		Proxy:     baseProxy,
		TargetURL: target,
		ActionURL: action,

		servertcp: &dns.Server{
			Addr:         cfg.ListenAddr,
			Net:          "tcp",
			ReadTimeout:  cfg.Timeout,
			WriteTimeout: cfg.Timeout,
		},
		serverudp: &dns.Server{
			Addr:         cfg.ListenAddr,
			Net:          "udp",
			ReadTimeout:  cfg.Timeout,
			WriteTimeout: cfg.Timeout,
		},
		client: &dns.Client{
			Timeout:      cfg.Timeout,
			DialTimeout:  cfg.Timeout,
			ReadTimeout:  cfg.Timeout,
			WriteTimeout: cfg.Timeout,
		},
	}
	if p.TLSConfig != nil {
		p.servertcp.Net = "tcp-tls"
		p.servertcp.TLSConfig = p.TLSConfig
	}
	p.servertcp.Handler = p.getHandler(p.servertcp.Net)
	p.serverudp.Handler = p.getHandler(p.serverudp.Net)

	return p, nil
}

type Proxy struct {
	*base.Proxy

	TargetURL netip.AddrPort
	ActionURL netip.AddrPort

	// NOTE: servertcp is a tcp or tcp-tls server,
	// serverudp is a udp server, used only when servertcp is tcp
	// (not tcp-tls).
	servertcp *dns.Server
	serverudp *dns.Server
	client    *dns.Client
}

func (p *Proxy) Start() error {
	p.WG.Add(1)
	go p.servetcp()
	if p.TLSConfig == nil {
		p.WG.Add(1)
		go p.serveudp()
	}
	return nil
}

func (p *Proxy) Shutdown(ctx context.Context) error {
	p.Closing = true
	if err := p.servertcp.ShutdownContext(ctx); err != nil {
		return fmt.Errorf("can't shutdown tcp server: %w", err)
	}
	if p.TLSConfig == nil {
		if err := p.serverudp.ShutdownContext(ctx); err != nil {
			return fmt.Errorf("can't shutdown udp server: %w", err)
		}
	}

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
	ap netip.AddrPort,
	w dns.ResponseWriter,
	r *dns.Msg,
	logger zerolog.Logger,
) {
	rr, _, err := p.client.Exchange(r, ap.String())
	if err != nil {
		logger.Error().Err(err).Msg("Can't make proxy request")
		return
	}

	err = w.WriteMsg(rr)
	if err != nil {
		logger.Error().Err(err).Msg("Can't make proxy response")
	}
}

func (p *Proxy) processVerdict(
	w dns.ResponseWriter,
	r *dns.Msg,
	logger zerolog.Logger,
) {
	switch p.Config.FilterSettings.Action {
	case common.ActionProxy:
		p.proxyRequest(p.ActionURL, w, r, logger)
	case common.ActionDrop:
		// do nothing
	default:
		logger.Warn().Msg("Request was filtered, but action is none")
		p.proxyRequest(p.TargetURL, w, r, logger)
	}
}

func (p *Proxy) getHandler(t string) dns.HandlerFunc {
	return func(w dns.ResponseWriter, r *dns.Msg) {
		defer w.Close()

		from := base.NetAddrToNetipAddrPort(w.RemoteAddr())
		logger := p.Logger.With().
			Stringer("from", from.Addr()).
			Str("type", t).
			Logger()

		logRequest(r, logger)

		e := &wrapper.DNSRequest{
			Request: r,
			From:    from.Addr(),
		}
		if !p.RunFilters(e, logger) {
			p.processVerdict(w, r, logger)
			return
		}

		p.proxyRequest(p.TargetURL, w, r, logger)
	}
}

func (p *Proxy) servetcp() {
	defer p.WG.Done()
	err := p.servertcp.ListenAndServe()
	if err != nil {
		p.Logger.Fatal().Err(err).Msg("Unexpected tcp server error")
	}
}

func (p *Proxy) serveudp() {
	defer p.WG.Done()
	err := p.serverudp.ListenAndServe()
	if err != nil {
		p.Logger.Fatal().Err(err).Msg("Unexpected udp server error")
	}
}
