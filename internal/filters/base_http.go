package filters

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/database"
	"github.com/D00Movenok/BounceBack/internal/wrapper"
	malleable "github.com/D00Movenok/goMalleable"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
	"golang.org/x/exp/slices"
)

func NewMalleableFilter(
	_ *database.DB,
	_ FilterSet,
	cfg common.FilterConfig,
) (Filter, error) {
	var params MallebaleFilterParams

	err := mapstructure.Decode(cfg.Params, &params)
	if err != nil {
		return nil, fmt.Errorf("can't decode params: %w", err)
	}

	file, err := os.Open(params.Profile)
	if err != nil {
		return nil, fmt.Errorf("can't open profile: %w", err)
	}

	parsed, err := malleable.Parse(file)
	if err != nil {
		return nil, fmt.Errorf("can't parse profile: %w", err)
	}

	filter := &MallebaleFilter{
		path:    params.Profile,
		profile: parsed,
		exclude: make([]*regexp.Regexp, 0, len(params.Exclude)),
	}

	for _, r := range params.Exclude {
		var re *regexp.Regexp
		re, err = regexp.Compile(r)
		if err != nil {
			return nil, fmt.Errorf("can't compile regex: %w", err)
		}
		filter.exclude = append(filter.exclude, re)
	}

	return filter, nil
}

type MallebaleFilterParams struct {
	Profile string   `mapstructure:"profile"`
	Exclude []string `mapstructure:"exclude"`
}

type MallebaleFilter struct {
	path    string
	exclude []*regexp.Regexp
	profile *malleable.Profile
}

func (f *MallebaleFilter) Apply(
	e wrapper.Entity,
	logger zerolog.Logger,
) (bool, error) {
	var url *url.URL

	// verify useragent
	allow, err := f.verifyUserAgent(e)
	if err != nil {
		return false, fmt.Errorf("can't verify user-agent: %w", err)
	}
	if !allow {
		return true, nil
	}

	// exlude pathes
	url, err = e.GetURL()
	if err != nil {
		return false, fmt.Errorf("can't get url: %w", err)
	}
	for _, e := range f.exclude {
		if e.MatchString(url.Path) {
			return false, nil
		}
	}

	// verify profiles
	for _, p := range f.profile.HTTPGet {
		allow, err = f.verifyHTTPProfile(
			e,
			logger,
			p.Verb,
			http.MethodGet,
			p.URI,
			p.Client.Parameters,
			p.Client.Headers,
			[][]malleable.Function{p.Client.Metadata},
		)
		if err != nil {
			return false, fmt.Errorf("can't verify get profile: %w", err)
		}
		if allow {
			return false, nil
		}
	}
	for _, p := range f.profile.HTTPPost {
		allow, err = f.verifyHTTPProfile(
			e,
			logger,
			p.Verb,
			http.MethodPost,
			p.URI,
			p.Client.Parameters,
			p.Client.Headers,
			[][]malleable.Function{p.Client.ID, p.Client.Output},
		)
		if err != nil {
			return false, fmt.Errorf("can't verify post profile: %w", err)
		}
		if allow {
			return false, nil
		}
	}

	// verify stager if exist
	if f.profile.HostStage {
		for _, p := range f.profile.HTTPStager {
			allow, err = f.verifyHTTPProfile(
				e,
				logger,
				http.MethodGet,
				http.MethodGet,
				append(p.URIx64, p.URIx86...),
				p.Client.Parameters,
				p.Client.Headers,
				[][]malleable.Function{{{Func: "uri-append"}}},
			)
			if err != nil {
				return false, fmt.Errorf(
					"can't verify stager profile: %w",
					err,
				)
			}
			if allow {
				return false, nil
			}
		}

		allow, err = f.verifyStagerURL(e)
		if err != nil {
			return false, fmt.Errorf("can't verify stager path: %w", err)
		}
		if allow {
			return false, nil
		}
	}

	return true, nil
}

func (f *MallebaleFilter) verifyUserAgent(e wrapper.Entity) (bool, error) {
	headers, err := e.GetHeaders()
	if err != nil {
		return false, fmt.Errorf("can't get headers: %w", err)
	}
	ua := headers["User-Agent"]

	// disallow blocked user-agents
	for _, b := range f.profile.HTTPConfig.BlockUserAgents {
		for _, u := range ua {
			if matchByMask(u, b) {
				return false, nil
			}
		}
	}

	// pass only allowed user-agents
	for _, a := range f.profile.HTTPConfig.AllowUserAgents {
		for _, u := range ua {
			if matchByMask(u, a) {
				return true, nil
			}
		}
	}
	if len(f.profile.HTTPConfig.AllowUserAgents) > 0 {
		return false, nil
	}

	// disabled, may cause download problems
	// if f.profile.UserAgent != "" {
	// 	if !slices.Contains(ua, f.profile.UserAgent) {
	// 		return false, nil
	// 	}
	// }

	return true, nil
}

func (f *MallebaleFilter) verifyHTTPProfile(
	e wrapper.Entity,
	logger zerolog.Logger,
	v string,
	dv string,
	u malleable.URIs,
	p []malleable.Parameter,
	h []malleable.Header,
	transforms [][]malleable.Function,
) (bool, error) {
	var (
		uriTransforms        []malleable.Function
		parametersTransforms []malleable.Function
		headersTransforms    []malleable.Function
		bodyTransforms       []malleable.Function
	)

	for _, transform := range transforms {
		last := transform[len(transform)-1]
		switch last.Func {
		case "header":
			headersTransforms = transform
		case "parameter":
			parametersTransforms = transform
		case "print":
			bodyTransforms = transform
		case "uri-append":
			uriTransforms = transform
		default:
			return false, errors.New("unknown transform: " + last.Func)
		}
	}

	// verify method
	ok, err := f.verifyMethod(e, v, dv)
	if err != nil {
		return false, fmt.Errorf("can't verify method: %w", err)
	}
	if !ok {
		return false, nil
	}

	// verify uri
	ok, err = f.verifyURI(e, logger, u, uriTransforms)
	if err != nil {
		return false, fmt.Errorf("can't verify uri: %w", err)
	}
	if !ok {
		return false, nil
	}

	// verify parameters
	ok, err = f.verifyParameters(e, logger, p, parametersTransforms)
	if err != nil {
		return false, fmt.Errorf("can't verify parameters: %w", err)
	}
	if !ok {
		return false, nil
	}

	// verify headers
	ok, err = f.verifyHeaders(e, logger, h, headersTransforms)
	if err != nil {
		return false, fmt.Errorf("can't verify headers: %w", err)
	}
	if !ok {
		return false, nil
	}

	// verify body if exist
	if bodyTransforms != nil {
		ok, err = f.verifyBody(e, logger, bodyTransforms)
		if err != nil {
			return false, fmt.Errorf("can't verify body: %w", err)
		}
		if !ok {
			return false, nil
		}
	}

	return true, nil
}

func (f *MallebaleFilter) verifyMethod(
	e wrapper.Entity,
	v string,
	dv string,
) (bool, error) {
	m, err := e.GetMethod()
	if err != nil {
		return false, fmt.Errorf("can't get method: %w", err)
	}

	// verify method
	if m != v && dv == "" && m != dv {
		return false, nil
	}

	return true, nil
}

func (f *MallebaleFilter) verifyURI(
	e wrapper.Entity,
	logger zerolog.Logger,
	uris malleable.URIs,
	transforms []malleable.Function,
) (bool, error) {
	var found bool

	url, err := e.GetURL()
	if err != nil {
		return false, fmt.Errorf("can't get url: %w", err)
	}

	for _, uri := range uris {
		// if metadata appended to uri
		hasAppended := strings.HasPrefix(url.Path, uri) && transforms != nil
		onlyURI := uri == url.Path && transforms == nil

		if onlyURI || hasAppended {
			found = true
			if hasAppended {
				found = f.verifyDecoding(
					[]byte(url.Path[len(uri):]),
					logger,
					transforms,
				)
			}
			if found {
				return true, nil
			}
		}
	}

	return false, nil
}

func (f *MallebaleFilter) verifyParameters(
	e wrapper.Entity,
	logger zerolog.Logger,
	parameters []malleable.Parameter,
	transforms []malleable.Function,
) (bool, error) {
	url, err := e.GetURL()
	if err != nil {
		return false, fmt.Errorf("can't get url: %w", err)
	}

	v := url.Query()
	for _, param := range parameters {
		if v.Get(param.Name) != param.Value {
			return false, nil
		}
	}

	// if metadata in param
	if transforms != nil {
		p := transforms[len(transforms)-1].Args[0]
		return f.verifyDecoding([]byte(v.Get(p)), logger, transforms), nil
	}

	return true, nil
}

func (f *MallebaleFilter) verifyHeaders(
	e wrapper.Entity,
	logger zerolog.Logger,
	pheaders []malleable.Header,
	transforms []malleable.Function,
) (bool, error) {
	headers, err := e.GetHeaders()
	if err != nil {
		return false, fmt.Errorf("can't get headers: %w", err)
	}

	for _, h := range pheaders {
		header, ok := headers[h.Name]
		if !ok || !slices.Contains(header, h.Value) {
			return false, nil
		}
	}

	// if metadata in header
	if transforms != nil {
		h := transforms[len(transforms)-1].Args[0]
		header, ok := headers[h]
		if ok {
			for _, h := range header {
				found := f.verifyDecoding([]byte(h), logger, transforms)
				if found {
					return true, nil
				}
			}
		}
		return false, nil
	}

	return true, nil
}

func (f *MallebaleFilter) verifyBody(
	e wrapper.Entity,
	logger zerolog.Logger,
	transforms []malleable.Function,
) (bool, error) {
	body, err := e.GetBody()
	if err != nil {
		return false, fmt.Errorf("can't get body: %w", err)
	}
	return f.verifyDecoding(body, logger, transforms), nil
}

func (f *MallebaleFilter) verifyStagerURL(e wrapper.Entity) (bool, error) {
	p, err := e.GetURL()
	if err != nil {
		return false, fmt.Errorf("can't get path: %w", err)
	}

	_, path := path.Split(p.Path)
	cs := checksum8([]byte(path))
	return cs == 92 || cs == 93, nil
}

func (f *MallebaleFilter) verifyDecoding(
	data []byte,
	logger zerolog.Logger,
	transforms []malleable.Function,
) bool {
	var (
		err error
		n   int
	)

	d := data
	t := transforms[:len(transforms)-1]
	for i := len(t) - 1; i >= 0; i-- {
		t := t[i]
		switch t.Func {
		case "append":
			if !(len(t.Args) == 1 && bytes.HasSuffix(d, []byte(t.Args[0]))) {
				return false
			}
			d = d[:len(d)-len(t.Args[0])]
		case "prepend":
			if !(len(t.Args) == 1 && bytes.HasPrefix(d, []byte(t.Args[0]))) {
				return false
			}
			d = d[len(t.Args[0]):]
		case "base64":
			n, err = base64.StdEncoding.Decode(d, d)
			if err != nil {
				return false
			}
			d = d[:n]
		case "base64url":
			n, err = base64.RawURLEncoding.Decode(d, d)
			if err != nil {
				return false
			}
			d = d[:n]
		case "mask":
			if len(d) < 5 { //nolint:gomnd // 4 byte key + atleast 1 byte data
				return false
			}
			d = xorDecrypt(d[:4], d[4:])
		case "netbios", "netbiosu":
			d, err = netbiosDecode(d)
			if err != nil {
				return false
			}
		default:
			logger.Error().Msgf("Unknown encoding: %s", t.Func)
		}
	}
	return len(data) > 0
}

func (f *MallebaleFilter) String() string {
	return fmt.Sprintf(
		"Malleable(profile=%s, exclude=%s)",
		f.path,
		FormatStringerSlice(f.exclude),
	)
}
