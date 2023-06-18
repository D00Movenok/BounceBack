package wrapper

import (
	"errors"
	"net/http"
	"net/netip"
	"net/url"
)

var (
	ErrNotSupported = errors.New("not supported")
)

type Entity interface {
	GetIP() netip.Addr
	GetRaw() ([]byte, error)

	// HTTP
	GetBody() ([]byte, error)
	GetCookies() ([]*http.Cookie, error)
	GetHeaders() (map[string][]string, error)
	GetURL() (*url.URL, error)
	GetMethod() (string, error)
}
