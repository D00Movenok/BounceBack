package wrapper

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/rs/zerolog/log"
)

// Request is a wrapper around http.Request implementing Entity interface.
// It's expected that Request.Body is already wrapped with BodyReader.
type Request struct {
	Request *http.Request
}

func (r Request) GetBody() ([]byte, error) {
	defer r.resetBody()
	buf, err := io.ReadAll(r.Request.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}
	return buf, nil
}

func (r Request) GetCookies() ([]*http.Cookie, error) {
	return r.Request.Cookies(), nil
}

func (r Request) GetHeaders() (map[string][]string, error) {
	return r.Request.Header, nil
}

func (r Request) GetURL() (*url.URL, error) {
	return r.Request.URL, nil
}

func (r Request) GetRaw() ([]byte, error) {
	defer r.resetBody()
	data, err := httputil.DumpRequest(r.Request, true)
	if err != nil {
		return nil, fmt.Errorf("dumping request: %w", err)
	}
	return data, err
}

func (r Request) resetBody() {
	if err := r.Request.Body.Close(); err != nil {
		log.Error().Err(err).Msg("Error resetting request body")
	}
}
