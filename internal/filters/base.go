package filters

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/database"
	"github.com/D00Movenok/BounceBack/internal/wrapper"
	"github.com/D00Movenok/BounceBack/pkg/ipapico"
	"github.com/D00Movenok/BounceBack/pkg/ipapicom"
	malleable "github.com/D00Movenok/goMalleable"
	badger "github.com/dgraph-io/badger/v3"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

func NewIPFilter(_ *database.DB, _ FilterSet, cfg common.FilterConfig) (Filter, error) {
	var (
		params IPFilterParams
		subnet netip.Prefix
		ip     netip.Addr
	)

	err := mapstructure.Decode(cfg.Params, &params)
	if err != nil {
		return nil, fmt.Errorf("can't decode params: %w", err)
	}

	filter := &IPFilter{
		path: params.Path,
	}

	file, err := os.Open(params.Path)
	if err != nil {
		return nil, fmt.Errorf("can't open ip list file: %w", err)
	}

	s := bufio.NewScanner(file)
	for s.Scan() {
		line := s.Text()
		line, _, _ = strings.Cut(line, "#") // remove comment
		line, _, _ = strings.Cut(line, " ") // remove space after
		isSubnet := strings.Contains(line, "/")
		if isSubnet {
			subnet, err = netip.ParsePrefix(line)
			filter.subnetBanlist = append(filter.subnetBanlist, subnet)
		} else if line != "" {
			ip, err = netip.ParseAddr(line)
			filter.ipBanlist = append(filter.ipBanlist, ip)
		}
		if err != nil {
			return nil, fmt.Errorf("can't parse ip/subnet: %w", err)
		}
	}

	return filter, nil
}

func NewTimeFilter(_ *database.DB, _ FilterSet, cfg common.FilterConfig) (Filter, error) {
	var params TimeParams
	err := mapstructure.Decode(cfg.Params, &params)
	if err != nil {
		return nil, fmt.Errorf("can't decode params: %w", err)
	}

	loc, err := time.LoadLocation(params.Location)
	if err != nil {
		return nil, fmt.Errorf("can't parse location: %w", err)
	}
	from, err := time.ParseInLocation("15:04", params.From, loc)
	if err != nil {
		return nil, fmt.Errorf("can't parse \"from\" time: %w", err)
	}
	to, err := time.ParseInLocation("15:04", params.To, loc)
	if err != nil {
		return nil, fmt.Errorf("can't parse \"to\" time: %w", err)
	}

	var (
		daysOfWeek = map[string]time.Weekday{
			"Sunday":    time.Sunday,
			"Monday":    time.Monday,
			"Tuesday":   time.Tuesday,
			"Wednesday": time.Wednesday,
			"Thursday":  time.Thursday,
			"Friday":    time.Friday,
			"Saturday":  time.Saturday,
		}
		days []time.Weekday
	)

	if len(params.Weekdays) == 0 {
		days = maps.Values(daysOfWeek)
	} else {
		for _, v := range params.Weekdays {
			d, ok := daysOfWeek[v]
			if !ok {
				return nil, fmt.Errorf("unknown day of week: %s", v)
			}
			days = append(days, d)
		}
	}

	filter := &TimeFilter{
		from:     from,
		to:       to,
		loc:      loc,
		weekdays: days,
	}

	return filter, nil
}

func NewGeolocationFilter(db *database.DB, _ FilterSet, cfg common.FilterConfig) (Filter, error) {
	var params GeoParams
	err := mapstructure.Decode(cfg.Params, &params)
	if err != nil {
		return nil, fmt.Errorf("can't decode params: %w", err)
	}

	filter := &GeoFilter{
		db:         db,
		geo:        make([]*GeoRegex, 0, len(params.Geolocations)),
		apicounter: atomic.NewInt32(0),
		// TODO: add clients with api keys
		ipapico:  ipapico.NewClient(),
		ipapicom: ipapicom.NewClient(),
	}

	var re *regexp.Regexp
	for _, gc := range params.Geolocations {
		gr := &GeoRegex{
			Organisation: make([]*regexp.Regexp, 0, len(gc.Organisation)),
			CountryCode:  make([]*regexp.Regexp, 0, len(gc.CountryCode)),
			Country:      make([]*regexp.Regexp, 0, len(gc.Country)),
			RegionCode:   make([]*regexp.Regexp, 0, len(gc.RegionCode)),
			Region:       make([]*regexp.Regexp, 0, len(gc.Region)),
			City:         make([]*regexp.Regexp, 0, len(gc.City)),
			Timezone:     make([]*regexp.Regexp, 0, len(gc.Timezone)),
			ASN:          make([]*regexp.Regexp, 0, len(gc.ASN)),
		}

		// iterate all fields of params.Geolocations, converts them to regexes and put to gr
		g := reflect.ValueOf(gc)
		for i := 0; i < g.NumField(); i++ {
			fn := g.Type().Field(i).Name
			for _, sre := range g.FieldByName(fn).Interface().([]string) {
				re, err = regexp.Compile(sre)
				if err != nil {
					return nil, fmt.Errorf("can't compile regex: %w", err)
				}
				reArr, _ := reflect.ValueOf(gr).Elem().FieldByName(fn).Addr().Interface().(*[]*regexp.Regexp)
				*reArr = append(*reArr, re)
			}
		}
		filter.geo = append(filter.geo, gr)
	}

	return filter, nil
}

func NewMalleableFilter(_ *database.DB, _ FilterSet, cfg common.FilterConfig) (Filter, error) {
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

type IPFilterParams struct {
	Path string `json:"list" mapstructure:"list"`
}

type IPFilter struct {
	path          string
	subnetBanlist []netip.Prefix
	ipBanlist     []netip.Addr
}

func (f *IPFilter) Apply(e wrapper.Entity, _ zerolog.Logger) (bool, error) {
	ip := e.GetIP()

	ok := slices.ContainsFunc(f.subnetBanlist, func(i netip.Prefix) bool {
		return i.Contains(ip)
	})
	if ok {
		return true, nil
	}

	ok = slices.ContainsFunc(f.ipBanlist, func(i netip.Addr) bool {
		return i.Compare(ip) == 0
	})
	if ok {
		return true, nil
	}

	return false, nil
}

func (f *IPFilter) String() string {
	return fmt.Sprintf("IP(list=%s)", f.path)
}

type TimeParams struct {
	From     string   `json:"from" mapstructure:"from"`
	To       string   `json:"to" mapstructure:"to"`
	Location string   `json:"timezone" mapstructure:"timezone"`
	Weekdays []string `json:"weekdays" mapstructure:"weekdays"`
}

type TimeFilter struct {
	from     time.Time
	to       time.Time
	loc      *time.Location
	weekdays []time.Weekday
}

func (f *TimeFilter) Apply(_ wrapper.Entity, _ zerolog.Logger) (bool, error) {
	n := time.Now().In(f.loc)

	d := n.Weekday()
	if !slices.Contains(f.weekdays, d) {
		return false, nil
	}

	now, _ := time.ParseInLocation("15:04", fmt.Sprintf("%02d:%02d", n.Hour(), n.Minute()), f.loc)
	fromLtTo := f.from.Before(f.to) && (now.Before(f.from) || now.After(f.to))
	fromGtTo := f.from.After(f.to) && (now.Before(f.from) && now.After(f.to))
	if fromLtTo || fromGtTo {
		return false, nil
	}

	return true, nil
}

func (f *TimeFilter) String() string {
	return fmt.Sprintf("Time(from=%02d:%02d, to=%02d:%02d, weekdays=%s, timezone=%s)",
		f.from.Hour(), f.from.Minute(), f.to.Hour(), f.to.Minute(), FormatStringerSlice(f.weekdays), f.loc.String())
}

// NOTE: GeoParam and GeoRegex must have same field names.
type GeoParam struct {
	Organisation []string `json:"organisation" mapstructure:"organisation"`
	CountryCode  []string `json:"country_code" mapstructure:"country_code"`
	Country      []string `json:"country" mapstructure:"country"`
	RegionCode   []string `json:"region_code" mapstructure:"region_code"`
	Region       []string `json:"region" mapstructure:"region"`
	City         []string `json:"city" mapstructure:"city"`
	Timezone     []string `json:"timezone" mapstructure:"timezone"`
	ASN          []string `json:"asn" mapstructure:"asn"`
}

type GeoParams struct {
	Geolocations []GeoParam `json:"geolocations" mapstructure:"geolocations"`
}

type GeoRegex struct {
	Organisation []*regexp.Regexp
	CountryCode  []*regexp.Regexp
	Country      []*regexp.Regexp
	RegionCode   []*regexp.Regexp
	Region       []*regexp.Regexp
	City         []*regexp.Regexp
	Timezone     []*regexp.Regexp
	ASN          []*regexp.Regexp
}

func (r *GeoRegex) String() string {
	return fmt.Sprintf(
		"geo(organisation=%s, country_code=%s, country=%s, region_code=%s, region=%s, city=%s, timezone=%s, asn=%s)",
		FormatStringerSlice(r.Organisation), FormatStringerSlice(r.CountryCode), FormatStringerSlice(r.Country),
		FormatStringerSlice(r.RegionCode), FormatStringerSlice(r.Region), FormatStringerSlice(r.City),
		FormatStringerSlice(r.Timezone), FormatStringerSlice(r.ASN),
	)
}

type GeoFilter struct {
	db         *database.DB
	geo        []*GeoRegex
	apicounter *atomic.Int32
	ipapico    ipapico.Client
	ipapicom   ipapicom.Client
}

func (f *GeoFilter) getGeoInfoByIP(ip string, logger zerolog.Logger) (*database.Geolocation, error) {
	geo, err := f.db.GetGeolocation(ip)
	if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
		return nil, fmt.Errorf("can't get cached geolocation: %w", err)
	}
	if geo != nil {
		return geo, nil
	}

	geo = &database.Geolocation{}
	switch f.apicounter.Inc() % 2 {
	case 0:
		var g *ipapico.Location
		// TODO: maybe add context with timeout
		g, err = f.ipapico.GetLocationForIP(context.Background(), ip)
		if err != nil && !errors.Is(err, ipapico.ErrReservedRange) {
			return nil, fmt.Errorf("can't get geolocation with ipapi.co: %w", err)
		}

		if g != nil {
			geo.Organisation = []string{g.Org}
			geo.CountryCode = g.Country
			geo.Country = g.CountryName
			geo.RegionCode = g.RegionCode
			geo.Region = g.Region
			geo.City = g.City
			geo.Timezone = g.Timezone
			geo.ASN = g.Asn
		}
	case 1:
		var g *ipapicom.Location
		// TODO: maybe add context with timeout
		g, err = f.ipapicom.GetLocationForIP(context.Background(), ip)
		if err != nil && !errors.Is(err, ipapicom.ErrReservedRange) {
			return nil, fmt.Errorf("can't get geolocation with ip-api.com: %w", err)
		}

		if g != nil {
			geo.Organisation = []string{g.Org, g.Isp, g.As}
			geo.CountryCode = g.CountryCode
			geo.Country = g.Country
			geo.RegionCode = g.Region
			geo.Region = g.RegionName
			geo.City = g.City
			geo.Timezone = g.Timezone
			geo.ASN, _, _ = strings.Cut(g.As, " ")
		}
	}

	logger.Debug().Any("geo", geo).Msg("New geo lookup")
	err = f.db.SaveGeolocation(ip, geo)
	if err != nil {
		return nil, fmt.Errorf("can't save geolocation: %w", err)
	}

	return geo, nil
}

func (f *GeoFilter) filterGeoByRegex(geo *database.Geolocation, gr *GeoRegex) bool {
	gf := reflect.ValueOf(gr).Elem()
	found := true
	// iterate field regexes and apply them on fields
	for i := 0; i < gf.NumField(); i++ {
		fn := gf.Type().Field(i).Name
		gv := reflect.ValueOf(geo).Elem().FieldByName(fn)
		regexes, _ := gf.FieldByName(fn).Interface().([]*regexp.Regexp)
		if gv.Len() == 0 || len(regexes) == 0 {
			continue
		}
		// find regex match of field fn.
		var m bool
		for _, re := range regexes {
			switch v := gv.Interface().(type) {
			case []string:
				for _, s := range v {
					m = re.MatchString(s)
					if m {
						break
					}
				}
			case string:
				m = re.MatchString(v)
			}
			if m {
				break
			}
		}
		// if field does not match, stop checking other fields with that GeoRegex arr element
		if !m {
			found = false
			break
		}
	}
	return found
}

func (f *GeoFilter) Apply(e wrapper.Entity, logger zerolog.Logger) (bool, error) {
	ip := e.GetIP().String()
	geo, err := f.getGeoInfoByIP(ip, logger)
	if err != nil {
		return false, fmt.Errorf("can't get geolocation info: %w", err)
	}
	for _, gr := range f.geo {
		m := f.filterGeoByRegex(geo, gr)
		if m {
			return true, nil
		}
	}
	return false, nil
}

func (f *GeoFilter) String() string {
	return fmt.Sprintf("Geo(geolocations=%s)", FormatStringerSlice(f.geo))
}

type MallebaleFilterParams struct {
	Profile string   `json:"profile" mapstructure:"profile"`
	Exclude []string `json:"exclude" mapstructure:"exclude"`
}

type MallebaleFilter struct {
	path    string
	exclude []*regexp.Regexp
	profile *malleable.Profile
}

func (f *MallebaleFilter) Apply(e wrapper.Entity, logger zerolog.Logger) (bool, error) {
	var (
		allow bool
		url   *url.URL
		err   error
	)

	// verify useragent
	allow, err = f.verifyUserAgent(e)
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
		allow, err = f.verifyHTTPProfile(e, logger, p.Verb, http.MethodGet, p.URI,
			p.Client.Parameters, p.Client.Headers, [][]malleable.Function{p.Client.Metadata})
		if err != nil {
			return false, fmt.Errorf("can't verify get profile: %w", err)
		}
		if allow {
			return false, nil
		}
	}
	for _, p := range f.profile.HTTPPost {
		allow, err = f.verifyHTTPProfile(e, logger, p.Verb, http.MethodPost, p.URI,
			p.Client.Parameters, p.Client.Headers, [][]malleable.Function{p.Client.ID, p.Client.Output})
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
			allow, err = f.verifyHTTPProfile(e, logger, http.MethodGet, http.MethodGet, append(p.URIx64, p.URIx86...),
				p.Client.Parameters, p.Client.Headers, [][]malleable.Function{{{Func: "uri-append"}}})
			if err != nil {
				return false, fmt.Errorf("can't verify stager profile: %w", err)
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

func (f *MallebaleFilter) verifyHTTPProfile(e wrapper.Entity, logger zerolog.Logger,
	v string, dv string, u malleable.URIs, p []malleable.Parameter,
	h []malleable.Header, transforms [][]malleable.Function) (bool, error) {
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

func (f *MallebaleFilter) verifyMethod(e wrapper.Entity, v string, dv string) (bool, error) {
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

func (f *MallebaleFilter) verifyURI(e wrapper.Entity, logger zerolog.Logger,
	uris malleable.URIs, transforms []malleable.Function) (bool, error) {
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
				found = f.verifyDecoding([]byte(url.Path[len(uri):]), logger, transforms)
			}
			if found {
				return true, nil
			}
		}
	}

	return false, nil
}

func (f *MallebaleFilter) verifyParameters(e wrapper.Entity, logger zerolog.Logger,
	parameters []malleable.Parameter, transforms []malleable.Function) (bool, error) {
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

func (f *MallebaleFilter) verifyHeaders(e wrapper.Entity, logger zerolog.Logger,
	pheaders []malleable.Header, transforms []malleable.Function) (bool, error) {
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

func (f *MallebaleFilter) verifyBody(e wrapper.Entity, logger zerolog.Logger,
	transforms []malleable.Function) (bool, error) {
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

func (f *MallebaleFilter) verifyDecoding(data []byte, logger zerolog.Logger,
	transforms []malleable.Function) bool {
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
	return fmt.Sprintf("Malleable(profile=%s, exclude=%s)", f.path, FormatStringerSlice(f.exclude))
}
