package wrapper

import (
	"net/http"
	"net/netip"
	"net/url"

	"github.com/miekg/dns"
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

	// DNS
	GetQuestions() ([]dns.Question, error)
}
