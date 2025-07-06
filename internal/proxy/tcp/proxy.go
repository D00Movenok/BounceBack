package tcp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/database"
	"github.com/D00Movenok/BounceBack/internal/proxy/base"
	"github.com/D00Movenok/BounceBack/internal/rules"
	"github.com/D00Movenok/BounceBack/internal/wrapper"
	"github.com/rs/zerolog"
)

const (
	ProxyType = "tcp"

	BufSize = 64 * 1024
)

var (
	AllowedActions = []string{
		common.RejectActionDrop,
		common.RejectActionNone,
	}
)

type Proxy struct {
	*base.Proxy

	IsTLS     bool
	TargetURL netip.AddrPort

	listener net.Listener
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

	p := &Proxy{
		Proxy: baseProxy,
	}

	scheme, ap, err := parseSchemeAddrPort(cfg.TargetAddr)
	if err != nil {
		return nil, fmt.Errorf("can't parse SchemeAddrPort: %w", err)
	}
	switch scheme {
	case "tcp":
		p.IsTLS = false
	case "tls":
		p.IsTLS = true
	default:
		return nil, &UnknownShemeError{scheme: scheme}
	}
	p.TargetURL = ap

	return p, nil
}

func (p *Proxy) Start() error {
	var err error
	if p.TLSConfig != nil {
		p.listener, err = tls.Listen("tcp", p.Config.ListenAddr, p.TLSConfig)
	} else {
		p.listener, err = net.Listen("tcp", p.Config.ListenAddr)
	}
	if err != nil {
		return fmt.Errorf("can't start listening: %w", err)
	}

	p.WG.Add(1)
	go p.serve()
	return nil
}

func (p *Proxy) Shutdown(ctx context.Context) error {
	p.Closing = true
	err := p.listener.Close()
	if err != nil {
		return fmt.Errorf("can't close listener: %w", err)
	}

	done := make(chan interface{}, 1)
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

// returns true if need return from func.
func (p *Proxy) processVerdict(
	src net.Conn,
	logger zerolog.Logger,
) bool {
	switch p.Config.RuleSettings.RejectAction {
	case common.RejectActionDrop:
		src.Close() //nolint:gosec // does not matter if error occurs
		return true
	default:
		logger.Warn().Msg("Request was filtered, but action is none")
		return false
	}
}

func (p *Proxy) oneSideHandler(
	e *wrapper.RawPacket,
	src net.Conn,
	dst net.Conn,
	logger zerolog.Logger,
	ingress bool,
) error {
	buf := make([]byte, BufSize)
	for {
		_ = src.SetDeadline(time.Now().Add(p.Config.Timeout))
		nr, err := src.Read(buf)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return fmt.Errorf("proxy connection read: %w", err)
			}
			break
		}
		data := buf[:nr]

		e.MU.Lock()
		if ingress {
			e.Content = append(e.Content, data...)
			if !p.RunFilters(e, logger) && p.processVerdict(src, logger) {
				e.MU.Unlock()
				return nil
			}
		} else {
			e.Content = []byte{}
		}
		e.MU.Unlock()

		var nw int
		_ = dst.SetDeadline(time.Now().Add(p.Config.Timeout))
		nw, err = dst.Write(data)
		if err != nil {
			return fmt.Errorf("proxy connection write: %w", err)
		}
		if nr != nw {
			return fmt.Errorf("proxy connection write: %w", io.ErrShortWrite)
		}
	}

	return nil
}

func (p *Proxy) handleConnection(src net.Conn) {
	defer p.WG.Done()
	defer src.Close()

	from := base.NetAddrToNetipAddrPort(src.RemoteAddr()).Addr().Unmap()
	logger := p.Logger.With().
		Stringer("from", from).
		Logger()

	logger.Info().Msg("New request")

	// first packet analysis so no data was read
	// TODO: drop filtered packets after SYN, not ACK.
	e := &wrapper.RawPacket{
		Content: []byte{},
		From:    from,
	}
	if !p.RunFilters(e, logger) && p.processVerdict(src, logger) {
		return
	}

	var (
		dst net.Conn
		err error
	)
	if p.IsTLS {
		dst, err = tls.Dial(
			"tcp",
			p.TargetURL.String(),
			p.TLSConfig,
		)
	} else {
		dst, err = net.Dial("tcp", p.TargetURL.String())
	}
	if err != nil {
		logger.Error().Err(err).Msg("Failed to connect to target")
		return
	}

	handler := func(
		src net.Conn,
		dst net.Conn,
		wg *sync.WaitGroup,
		ingress bool,
	) {
		defer wg.Done()
		err = p.oneSideHandler(
			e,
			src,
			dst,
			logger,
			ingress,
		)
		if err != nil && !base.IsConnectionClosed(err) {
			logger.Error().Err(err).Msg("Connection error")
		}

		// if not close both, ne side will wait until timeout
		src.Close() //nolint:gosec // does not matter if error occurs
		dst.Close() //nolint:gosec // does not matter if error occurs
	}

	wg := sync.WaitGroup{}
	wg.Add(2) //nolint:mnd // two connections
	go handler(src, dst, &wg, true)
	go handler(dst, src, &wg, false)
	wg.Wait()
}

func (p *Proxy) serve() {
	defer p.WG.Done()
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			if !p.Closing {
				p.Logger.Error().Err(err).Msg("Unexpected server error")
			}
			return
		}

		p.WG.Add(1)
		go p.handleConnection(conn)
	}
}
