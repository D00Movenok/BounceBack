package wrapper

import (
	"bytes"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"

	"github.com/miekg/dns"
)

// DNSRequest is a wrapper around dns.Msg implementing Entity interface.
type DNSRequest struct {
	Request *dns.Msg
	From    netip.Addr
}

func (r *DNSRequest) GetIP() netip.Addr {
	return r.From
}

// GetRaw return all questions in format:
// REQ_TYPE1 DOMAIN_NAME1
// REQ_TYPE2 DOMAIN_NAME2
// ...
func (r *DNSRequest) GetRaw() ([]byte, error) {
	var d bytes.Buffer
	for _, q := range r.Request.Question {
		_, err := d.WriteString(
			dns.TypeToString[q.Qtype] + " " + q.Name + "\n",
		)
		if err != nil {
			return nil, fmt.Errorf("can't add question name: %w", err)
		}
	}
	return d.Bytes(), nil
}

func (r *DNSRequest) GetBody() ([]byte, error) {
	return nil, ErrNotSupported
}

func (r *DNSRequest) GetCookies() ([]*http.Cookie, error) {
	return nil, ErrNotSupported
}

func (r *DNSRequest) GetHeaders() (map[string][]string, error) {
	return nil, ErrNotSupported
}

func (r *DNSRequest) GetURL() (*url.URL, error) {
	return nil, ErrNotSupported
}

func (r *DNSRequest) GetMethod() (string, error) {
	return "", ErrNotSupported
}

func (r *DNSRequest) GetQuestions() ([]dns.Question, error) {
	return r.Request.Question, nil
}
