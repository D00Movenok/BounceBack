package rules

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
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

// TODO: add unit tests for malleable rule.
func NewMalleableRule(
	_ *database.DB,
	_ RuleSet,
	cfg common.RuleConfig,
	_ common.Globals,
) (Rule, error) {
	var params MallebaleRuleParams

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

	rule := &MallebaleRule{
		path:    params.Profile,
		profile: parsed,
		exclude: make([]*regexp.Regexp, 0, len(params.Exclude)),
	}

	for _, r := range params.Exclude {
		var re *regexp.Regexp
		re, err = regexp.Compile(r)
		if err != nil {
			return nil, fmt.Errorf("can't compile regexp: %w", err)
		}
		rule.exclude = append(rule.exclude, re)
	}

	return rule, nil
}

type MallebaleRuleParams struct {
	Profile string   `mapstructure:"profile"`
	Exclude []string `mapstructure:"exclude"`
}

type MallebaleRule struct {
	path    string
	exclude []*regexp.Regexp
	profile *malleable.Profile
}

func (f *MallebaleRule) Prepare(
	_ wrapper.Entity,
	_ zerolog.Logger,
) error {
	return nil
}

func (f *MallebaleRule) Apply(
	e wrapper.Entity,
	logger zerolog.Logger,
) (bool, error) {
	var found bool

	// verify useragent lists
	allow, err := f.verifyUserAgentLists(e, logger)
	if err != nil {
		return false, fmt.Errorf("can't verify user-agent lists: %w", err)
	}
	if !allow {
		return true, nil
	}

	// exlude pathes
	found, err = f.isExcluded(e, logger)
	if err != nil {
		return false, fmt.Errorf("can't verify excluded paths: %w", err)
	}
	if found {
		return false, nil
	}

	// verify useragent
	allow, err = f.verifyUserAgent(e, logger)
	if err != nil {
		return false, fmt.Errorf("can't verify user-agent: %w", err)
	}
	if !allow {
		return true, nil
	}

	// verify profiles
	found, err = f.findProfile(e, logger)
	if err != nil {
		return false, fmt.Errorf("can't find profile: %w", err)
	}
	if found {
		return false, nil
	}

	logger.Debug().Msg("http-get/post/stager did not match")
	return true, nil
}

func (f *MallebaleRule) verifyUserAgentLists(
	e wrapper.Entity,
	logger zerolog.Logger,
) (bool, error) {
	headers, err := e.GetHeaders()
	if err != nil {
		return false, fmt.Errorf("can't get headers: %w", err)
	}
	ua := headers["User-Agent"]

	// disallow blocked user-agents
	for _, b := range f.profile.HTTPConfig.BlockUserAgents {
		for _, u := range ua {
			if matchByMask(u, b) {
				logger.Debug().Str("match", u).Msg("block_useragents match")
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
		logger.Debug().Any("match", ua).Msg("allow_useragents did not match")
		return false, nil
	}

	return true, nil
}

func (f *MallebaleRule) isExcluded(
	e wrapper.Entity,
	logger zerolog.Logger,
) (bool, error) {
	url, err := e.GetURL()
	if err != nil {
		return false, fmt.Errorf("can't get url: %w", err)
	}
	for _, e := range f.exclude {
		if e.MatchString(url.Path) {
			logger.Trace().Stringer("match", e).Msg("Exclude URL match")
			return true, nil
		}
	}
	return false, nil
}

func (f *MallebaleRule) verifyUserAgent(
	e wrapper.Entity,
	logger zerolog.Logger,
) (bool, error) {
	headers, err := e.GetHeaders()
	if err != nil {
		return false, fmt.Errorf("can't get headers: %w", err)
	}
	ua := headers["User-Agent"]

	if f.profile.UserAgent != "" && !slices.Contains(ua, f.profile.UserAgent) {
		logger.Debug().Any("match", ua).Msg("user_agent did not match")
		return false, nil
	}

	return true, nil
}

func (f *MallebaleRule) findProfile(
	e wrapper.Entity,
	logger zerolog.Logger,
) (bool, error) {
	for _, p := range f.profile.HTTPGet {
		pl := logger.With().
			Str("profile", p.Name).
			Str("type", "http-get").
			Logger()
		allow, err := f.verifyHTTPProfile(
			e,
			pl,
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
			pl.Trace().Msg("Match")
			return true, nil
		}
	}

	for _, p := range f.profile.HTTPPost {
		pl := logger.With().
			Str("profile", p.Name).
			Str("type", "http-post").
			Logger()
		allow, err := f.verifyHTTPProfile(
			e,
			pl,
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
			pl.Trace().Msg("Match")
			return true, nil
		}
	}

	if f.profile.HostStage {
		for _, p := range f.profile.HTTPStager {
			pl := logger.With().
				Str("profile", p.Name).
				Str("type", "http-stager").
				Logger()
			allow, err := f.verifyHTTPProfile(
				e,
				pl,
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
				pl.Trace().Msg("Match")
				return true, nil
			}
		}

		allow, err := f.verifyStagerURL(e)
		if err != nil {
			return false, fmt.Errorf("can't verify stager path: %w", err)
		}
		if allow {
			// Debug, because blues may use stage urls for C2 searching
			logger.Debug().Msg("Stager URL match")
			return true, nil
		}
	}

	return false, nil
}

func (f *MallebaleRule) verifyHTTPProfile(
	e wrapper.Entity,
	logger zerolog.Logger,
	method string,
	defaultMethod string,
	uri malleable.URIs,
	parameters []malleable.Parameter,
	headers []malleable.Header,
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
			return false, &UnknownTransformError{transform: last.Func}
		}
	}

	// verify method
	ok, err := f.verifyMethod(e, logger, method, defaultMethod)
	if err != nil {
		return false, fmt.Errorf("can't verify method: %w", err)
	}
	if !ok {
		return false, nil
	}

	// verify uri
	ok, err = f.verifyURI(e, logger, uri, uriTransforms)
	if err != nil {
		return false, fmt.Errorf("can't verify uri: %w", err)
	}
	if !ok {
		return false, nil
	}

	// verify parameters
	ok, err = f.verifyParameters(e, logger, parameters, parametersTransforms)
	if err != nil {
		return false, fmt.Errorf("can't verify parameters: %w", err)
	}
	if !ok {
		return false, nil
	}

	// verify headers
	ok, err = f.verifyHeaders(e, logger, headers, headersTransforms)
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

func (f *MallebaleRule) verifyMethod(
	e wrapper.Entity,
	logger zerolog.Logger,
	method string,
	defaultMethod string,
) (bool, error) {
	m, err := e.GetMethod()
	if err != nil {
		return false, fmt.Errorf("can't get method: %w", err)
	}

	wantMethod := defaultMethod
	if method != "" {
		wantMethod = method
	}

	// verify method
	if m != wantMethod {
		logger.Trace().
			Str("method", wantMethod).
			Msg("Method mismatch")
		return false, nil
	}

	return true, nil
}

func (f *MallebaleRule) verifyURI(
	e wrapper.Entity,
	logger zerolog.Logger,
	uris malleable.URIs,
	transforms []malleable.Function,
) (bool, error) {
	url, err := e.GetURL()
	if err != nil {
		return false, fmt.Errorf("can't get url: %w", err)
	}

	for _, uri := range uris {
		// if metadata appended to uri
		hasAppended := strings.HasPrefix(url.Path, uri) && transforms != nil
		onlyURI := uri == url.Path && transforms == nil

		if onlyURI {
			return true, nil
		}
		if hasAppended {
			found := f.verifyDecoding(
				[]byte(url.Path[len(uri):]),
				logger,
				transforms,
			)
			if found {
				return true, nil
			}
			logger.Trace().Str("uri", url.Path).Msg("Can't decode URI")
		}
	}

	if transforms == nil {
		logger.Trace().Any("uris", uris).Msg("URI mismatch")
	}

	return false, nil
}

func (f *MallebaleRule) verifyParameters(
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
			logger.Trace().
				Dict(
					"parameter",
					zerolog.Dict().
						Str("key", param.Name).
						Str("value", param.Value),
				).
				Msg("Query parameter mismatch")
			return false, nil
		}
	}

	// if metadata in param
	if transforms != nil {
		p := transforms[len(transforms)-1].Args[0]
		v := v.Get(p)
		found := f.verifyDecoding([]byte(v), logger, transforms)
		if !found {
			logger.Trace().
				Dict(
					"parameter",
					zerolog.Dict().
						Str("key", p).
						Str("value", v),
				).
				Msg("Can't decode query parameter")
		}
		return found, nil
	}

	return true, nil
}

func (f *MallebaleRule) verifyHeaders(
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
		canonical := http.CanonicalHeaderKey(h.Name)
		header, ok := headers[canonical]
		if !ok || !slices.ContainsFunc(header, func(e string) bool {
			return strings.EqualFold(e, h.Value)
		}) {
			logger.Trace().
				Dict(
					"header",
					zerolog.Dict().
						Str("key", canonical).
						Str("value", h.Value),
				).
				Msg("Header mismatch")
			return false, nil
		}
	}

	// if metadata in header
	if transforms != nil {
		h := transforms[len(transforms)-1].Args[0]
		va, ok := headers[h]
		if ok {
			for _, v := range va {
				found := f.verifyDecoding([]byte(v), logger, transforms)
				if found {
					return true, nil
				}
			}
		}
		logger.Trace().
			Dict(
				"header",
				zerolog.Dict().
					Str("key", h).
					Any("value", va),
			).
			Msg("Can't decode header")
		return false, nil
	}

	return true, nil
}

func (f *MallebaleRule) verifyBody(
	e wrapper.Entity,
	logger zerolog.Logger,
	transforms []malleable.Function,
) (bool, error) {
	body, err := e.GetBody()
	if err != nil {
		return false, fmt.Errorf("can't get body: %w", err)
	}
	found := f.verifyDecoding(body, logger, transforms)
	if !found {
		logger.Trace().
			Bytes("body", body).
			Msg("Can't decode body")
	}
	return found, nil
}

func (f *MallebaleRule) verifyStagerURL(e wrapper.Entity) (bool, error) {
	p, err := e.GetURL()
	if err != nil {
		return false, fmt.Errorf("can't get path: %w", err)
	}

	_, path := path.Split(p.Path)
	cs := checksum8([]byte(path))
	return cs == 92 || cs == 93, nil
}

func (f *MallebaleRule) verifyDecoding(
	data []byte,
	logger zerolog.Logger,
	transforms []malleable.Function,
) bool {
	var (
		err error
		n   int
	)

	d := data
	t := transforms[:len(transforms)-1] // last transform is where to store data
	for i := len(t) - 1; i >= 0; i-- {
		t := t[i]
		switch t.Func {
		case "append":
			if !(len(t.Args) == 1 && bytes.HasSuffix(d, []byte(t.Args[0]))) {
				logger.Trace().
					Str("func", t.Func).
					Str("arg", t.Args[0]).
					Msg("Can't decode")
				return false
			}
			d = d[:len(d)-len(t.Args[0])]
		case "prepend":
			if !(len(t.Args) == 1 && bytes.HasPrefix(d, []byte(t.Args[0]))) {
				logger.Trace().
					Str("func", t.Func).
					Str("arg", t.Args[0]).
					Msg("Can't decode")
				return false
			}
			d = d[len(t.Args[0]):]
		case "base64":
			n, err = base64.StdEncoding.Decode(d, d)
			if err != nil {
				logger.Trace().
					Str("func", t.Func).
					Msg("Can't decode")
				return false
			}
			d = d[:n]
		case "base64url":
			n, err = base64.RawURLEncoding.Decode(d, d)
			if err != nil {
				logger.Trace().
					Str("func", t.Func).
					Msg("Can't decode")
				return false
			}
			d = d[:n]
		case "mask":
			if len(d) < 5 { //nolint:gomnd // 4 byte key + atleast 1 byte data
				logger.Trace().
					Str("func", t.Func).
					Msg("Can't decode")
				return false
			}
			d = xorDecrypt(d[:4], d[4:])
		case "netbios", "netbiosu":
			d, err = netbiosDecode(d, t.Func == "netbios")
			if err != nil {
				logger.Trace().
					Str("func", t.Func).
					Msg("Can't decode")
				return false
			}
		default:
			logger.Error().Msgf("Unknown encoding: %s", t.Func)
		}
	}
	return len(data) > 0
}

func (f *MallebaleRule) String() string {
	return fmt.Sprintf(
		"Malleable(profile=%s, exclude=%s)",
		f.path,
		common.FormatStringerSlice(f.exclude),
	)
}
