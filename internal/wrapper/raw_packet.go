package wrapper

import (
	"net/http"
	"net/netip"
	"net/url"
	"sync"
)

// RawPacket is a wrapper around raw data (e.g. tcp packet)
// implementing Entity interface.
type RawPacket struct {
	Content []byte
	From    netip.Addr
	MU      sync.Mutex
}

func (p *RawPacket) GetIP() netip.Addr {
	return p.From
}

func (p *RawPacket) GetRaw() ([]byte, error) {
	dst := make([]byte, len(p.Content))
	copy(dst, p.Content)
	return dst, nil
}

func (p *RawPacket) GetBody() ([]byte, error) {
	dst := make([]byte, len(p.Content))
	copy(dst, p.Content)
	return dst, nil
}

func (p *RawPacket) GetCookies() ([]*http.Cookie, error) {
	return nil, ErrNotSupported
}

func (p *RawPacket) GetHeaders() (map[string][]string, error) {
	return nil, ErrNotSupported
}

func (p *RawPacket) GetURL() (*url.URL, error) {
	return nil, ErrNotSupported
}

func (p *RawPacket) GetMethod() (string, error) {
	return "", ErrNotSupported
}
