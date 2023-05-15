package wrapper

import (
	"errors"
	"net/http"
	"net/url"
)

var (
	ErrNotSupported = errors.New("not supported")
)

type Entity interface {
	GetCookies() ([]*http.Cookie, error)
	GetHeaders() (map[string][]string, error)
	GetURL() (*url.URL, error)
	GetBody() ([]byte, error)
	GetRaw() ([]byte, error)
}
