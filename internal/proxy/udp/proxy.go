package udp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
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
	ProxyType = "udp"

	BufSize = 64 * 1024
)

var (
	AllowedActions = []string{
		common.RejectActionDrop,
		common.RejectActionNone,
	}
)

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

	if p.TLSConfig != nil {
		return nil, base.ErrTLSUnsupported
	}

	ap, err := netip.ParseAddrPort(cfg.TargetAddr)
	if err != nil {
		return nil, fmt.Errorf("can't parse AddrPort: %w", err)
	}
	p.TargetURL = ap

	return p, nil
}

type Proxy struct {
	*base.Proxy

	TargetURL netip.AddrPort

	connMap  sync.Map
	listener *net.UDPConn
}

func (p *Proxy) Start() error {
	host, portStr, err := net.SplitHostPort(p.Config.ListenAddr)
	if err != nil {
		return fmt.Errorf("splitting hostport: %w", err)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("converting port to int: %w", err)
	}

	addr := &net.UDPAddr{
		Port: port,
		IP:   net.ParseIP(host),
	}

	p.listener, err = net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("can't run listen: %w", err)
	}

	p.WG.Add(1)
	go p.serve()
	return nil
}

func (p *Proxy) Shutdown(ctx context.Context) error {
	p.Closing = true
	if err := p.listener.Close(); err != nil {
		return fmt.Errorf("can't close listener: %w", err)
	}

	done := make(chan interface{}, 1)
	go func() {
		p.connMap.Range(func(_, value any) bool {
			c, _ := value.(*Connection)
			c.Close()
			return true
		})
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
	c *Connection,
	logger zerolog.Logger,
) bool {
	switch p.Config.RuleSettings.RejectAction {
	case common.RejectActionDrop:
		c.Close()
		p.connMap.Delete(c.String())
		return true
	default:
		logger.Warn().Msg("Request was filtered, but action is none")
		return false
	}
}

func (p *Proxy) replyLoop(
	c *Connection,
	logger zerolog.Logger,
) {
	defer func() {
		c.Close()
		p.connMap.Delete(c.String())
		p.WG.Done()
	}()

	conn := *c.Dst
	buf := make([]byte, BufSize)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(p.Config.Timeout))
		read, err := conn.Read(buf)
		if err != nil {
			netErr := new(net.Error)
			if errors.As(err, netErr); (*netErr).Timeout() {
				return
			}

			logger.Error().Err(err).Msg("Connection error")
			return
		}

		_ = p.listener.SetWriteDeadline(time.Now().Add(p.Config.Timeout))
		_, err = p.listener.WriteToUDP(buf[:read], c.Src)
		if err != nil {
			logger.Error().Err(err).Msg("Proxy connection write")
			return
		}
	}
}

func (p *Proxy) handleConnection(src *net.UDPAddr, data []byte) {
	defer p.WG.Done()

	from := src.AddrPort().Addr().Unmap()
	logger := p.Logger.With().
		Stringer("from", from).
		Logger()

	c := newConnection(src)
	v, exist := p.connMap.LoadOrStore(c.String(), c)
	if exist {
		c, _ = v.(*Connection)
	} else {
		logger.Info().Msg("New request")
	}

	e := &wrapper.RawPacket{
		Content: data,
		From:    from,
	}
	if !p.RunFilters(e, logger) && p.processVerdict(c, logger) {
		return
	}

	if !exist {
		dst, err := net.Dial("udp", p.TargetURL.String())
		if err != nil {
			logger.Error().Err(err).Msg("Failed to connect to target")
			return
		}
		c.Dst = &dst

		p.WG.Add(1)
		go p.replyLoop(c, logger)
	}

	_ = (*c.Dst).SetWriteDeadline(time.Now().Add(p.Config.Timeout))
	_, ew := (*c.Dst).Write(data)
	if ew != nil {
		logger.Error().Err(ew).Msg("Proxy connection write")
		return
	}
}

func (p *Proxy) serve() {
	defer p.WG.Done()
	buf := make([]byte, BufSize)
	for {
		read, from, err := p.listener.ReadFromUDP(buf)
		if err != nil {
			if !p.Closing {
				p.Logger.Error().Err(err).Msg("Unexpected server error")
			}
			return
		}

		p.WG.Add(1)
		go p.handleConnection(from, buf[:read])
	}
}
