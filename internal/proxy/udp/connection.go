package udp

import (
	"fmt"
	"net"

	"github.com/D00Movenok/BounceBack/internal/proxy/base"
)

func newConnection(src *net.UDPAddr) *Connection {
	return &Connection{
		Src: src,
	}
}

type Connection struct {
	Src *net.UDPAddr
	Dst *net.Conn
}

func (c Connection) Close() error {
	err := (*c.Dst).Close()
	if err != nil && !base.IsConnectionClosed(err) {
		return fmt.Errorf("closing connection: %w", err)
	}
	return nil
}

func (c Connection) String() string {
	return c.Src.AddrPort().String()
}
