package rules

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/database"
	"github.com/D00Movenok/BounceBack/internal/wrapper"
	"github.com/D00Movenok/BounceBack/pkg/ipapico"
	"github.com/D00Movenok/BounceBack/pkg/ipapicom"
	badger "github.com/dgraph-io/badger/v3"
	"github.com/miekg/dns"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.uber.org/atomic"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

func NewRegexpRule(
	_ *database.DB,
	_ RuleSet,
	cfg common.RuleConfig,
	_ common.Globals,
) (Rule, error) {
	var params RegexpParams

	err := mapstructure.Decode(cfg.Params, &params)
	if err != nil {
		return nil, fmt.Errorf("can't decode params: %w", err)
	}

	rule := &RegexpRule{
		path: params.Path,
	}

	rule.list, err = getRegexpList(params.Path)
	if err != nil {
		return nil, fmt.Errorf("can't create regexp list: %w", err)
	}

	return rule, nil
}

func NewIPRule(
	_ *database.DB,
	_ RuleSet,
	cfg common.RuleConfig,
	_ common.Globals,
) (Rule, error) {
	var (
		params IPRuleParams
		subnet netip.Prefix
		ip     netip.Addr
	)

	err := mapstructure.Decode(cfg.Params, &params)
	if err != nil {
		return nil, fmt.Errorf("can't decode params: %w", err)
	}

	rule := &IPRule{
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
		line = strings.TrimSpace(line)      // trim spaces
		isSubnet := strings.Contains(line, "/")
		if isSubnet {
			subnet, err = netip.ParsePrefix(line)
			rule.subnetBanlist = append(rule.subnetBanlist, subnet)
		} else if line != "" {
			ip, err = netip.ParseAddr(line)
			rule.ipBanlist = append(rule.ipBanlist, ip)
		}
		if err != nil {
			return nil, fmt.Errorf("can't parse ip/subnet: %w", err)
		}
	}

	// sort and remove equal elements for ipBanlist
	slices.SortFunc(rule.ipBanlist, func(e1 netip.Addr, e2 netip.Addr) int {
		return e1.Compare(e2)
	})
	rule.ipBanlist = slices.Compact(rule.ipBanlist)

	// sort and remove equal elements for subnetBanlist
	// TODO: update with compare func when it will be added
	// https://github.com/golang/go/issues/61642
	slices.SortFunc(
		rule.subnetBanlist,
		func(e1 netip.Prefix, e2 netip.Prefix) int {
			return e1.Masked().Addr().Compare(e2.Masked().Addr())
		},
	)
	rule.subnetBanlist = slices.CompactFunc(
		rule.subnetBanlist,
		func(e1 netip.Prefix, e2 netip.Prefix) bool {
			return e1.Overlaps(e2)
		},
	)

	return rule, nil
}

func NewTimeRule(
	_ *database.DB,
	_ RuleSet,
	cfg common.RuleConfig,
	_ common.Globals,
) (Rule, error) {
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
				return nil, &UnknownDayOfWeekError{day: v}
			}
			days = append(days, d)
		}
	}

	rule := &TimeRule{
		from:     from,
		to:       to,
		loc:      loc,
		weekdays: days,
	}

	return rule, nil
}

func NewGeolocationRule(
	db *database.DB,
	_ RuleSet,
	cfg common.RuleConfig,
	gloals common.Globals,
) (Rule, error) {
	var params GeoParams
	err := mapstructure.Decode(cfg.Params, &params)
	if err != nil {
		return nil, fmt.Errorf("can't decode params: %w", err)
	}

	var ipapicoClient ipapico.Client
	if gloals.IPApiCoKey != "" {
		ipapicoClient = ipapico.NewClientWithAPIKey(gloals.IPApiCoKey)
	} else {
		ipapicoClient = ipapico.NewClient()
	}

	var ipapicomClient ipapicom.Client
	if gloals.IPApiCoKey != "" {
		ipapicomClient = ipapicom.NewClientWithAPIKey(gloals.IPApiCoKey)
	} else {
		ipapicomClient = ipapicom.NewClient()
	}

	rule := &GeoRule{
		db:         db,
		path:       params.Path,
		geo:        make([]*GeoRegexp, 0, len(params.Geolocations)),
		apicounter: atomic.NewInt32(0),
		ipapico:    ipapicoClient,
		ipapicom:   ipapicomClient,
	}

	if params.Path != "" {
		rule.list, err = getRegexpList(params.Path)
		if err != nil {
			return nil, fmt.Errorf("can't create regexp list: %w", err)
		}
	}

	var re *regexp.Regexp
	for _, gc := range params.Geolocations {
		gr := &GeoRegexp{
			Organisation: make([]*regexp.Regexp, 0, len(gc.Organisation)),
			CountryCode:  make([]*regexp.Regexp, 0, len(gc.CountryCode)),
			Country:      make([]*regexp.Regexp, 0, len(gc.Country)),
			RegionCode:   make([]*regexp.Regexp, 0, len(gc.RegionCode)),
			Region:       make([]*regexp.Regexp, 0, len(gc.Region)),
			City:         make([]*regexp.Regexp, 0, len(gc.City)),
			Timezone:     make([]*regexp.Regexp, 0, len(gc.Timezone)),
			ASN:          make([]*regexp.Regexp, 0, len(gc.ASN)),
		}

		// iterate all fields of params.Geolocations,
		// converts them to regexps and put to gr
		g := reflect.ValueOf(gc)
		for i := 0; i < g.NumField(); i++ {
			fn := g.Type().Field(i).Name
			for _, sre := range g.FieldByName(fn).Interface().([]string) {
				re, err = regexp.Compile(sre)
				if err != nil {
					return nil, fmt.Errorf("can't compile regexp: %w", err)
				}
				reArr, _ := reflect.
					ValueOf(gr).
					Elem().
					FieldByName(fn).
					Addr().
					Interface().(*[]*regexp.Regexp)
				*reArr = append(*reArr, re)
			}
		}
		rule.geo = append(rule.geo, gr)
	}

	return rule, nil
}

func NewReverseLookupRule(
	db *database.DB,
	_ RuleSet,
	cfg common.RuleConfig,
	_ common.Globals,
) (Rule, error) {
	var (
		params ReverseLookupParams
		dns    netip.AddrPort
	)

	err := mapstructure.Decode(cfg.Params, &params)
	if err != nil {
		return nil, fmt.Errorf("can't decode params: %w", err)
	}

	dns, err = netip.ParseAddrPort(params.DNS)
	if err != nil {
		return nil, fmt.Errorf("dns addr is invalid: %w", err)
	}

	rule := &ReverseLookupRule{
		db:   db,
		path: params.Path,
		dns:  dns,
	}

	rule.list, err = getRegexpList(params.Path)
	if err != nil {
		return nil, fmt.Errorf("can't create regexp list: %w", err)
	}

	return rule, nil
}

type RegexpParams struct {
	Path string `mapstructure:"list"`
}

type RegexpRule struct {
	path string
	list []*regexp.Regexp
}

func (f *RegexpRule) Prepare(
	_ wrapper.Entity,
	_ zerolog.Logger,
) error {
	return nil
}

func (f RegexpRule) Apply(
	e wrapper.Entity,
	logger zerolog.Logger,
) (bool, error) {
	raw, err := e.GetRaw()
	if err != nil {
		return false, fmt.Errorf("can't get raw packet: %w", err)
	}
	for _, r := range f.list {
		if r.Match(raw) {
			logger.Debug().Stringer("match", r).Msg("Regexp match")
			return true, nil
		}
	}
	return false, nil
}

func (f RegexpRule) String() string {
	return fmt.Sprintf("Regexp(list=%s)", f.path)
}

type IPRuleParams struct {
	Path string `mapstructure:"list"`
}

type IPRule struct {
	path          string
	subnetBanlist []netip.Prefix
	ipBanlist     []netip.Addr
}

func (f *IPRule) Prepare(
	_ wrapper.Entity,
	_ zerolog.Logger,
) error {
	return nil
}

func (f *IPRule) Apply(
	e wrapper.Entity,
	logger zerolog.Logger,
) (bool, error) {
	ip := e.GetIP()

	// search ip in subnetBanlist
	// TODO: use Compare func when
	// https://github.com/golang/go/issues/61642
	i, found := slices.BinarySearchFunc(
		f.subnetBanlist,
		ip,
		func(e1 netip.Prefix, e2 netip.Addr) int {
			if e1.Contains(e2) {
				return 0
			}
			return e1.Masked().Addr().Compare(e2)
		},
	)
	if found {
		logger.Debug().Stringer("match", f.subnetBanlist[i]).Msg("Subnet match")
		return true, nil
	}

	// search ip in ipBanlist
	i, found = slices.BinarySearchFunc(
		f.ipBanlist,
		ip,
		func(e1 netip.Addr, e2 netip.Addr) int {
			return e1.Compare(e2)
		},
	)
	if found {
		logger.Debug().Stringer("match", f.ipBanlist[i]).Msg("IP match")
		return true, nil
	}

	return false, nil
}

func (f *IPRule) String() string {
	return fmt.Sprintf("IP(list=%s)", f.path)
}

type TimeParams struct {
	From     string   `mapstructure:"from"`
	To       string   `mapstructure:"to"`
	Location string   `mapstructure:"timezone"`
	Weekdays []string `mapstructure:"weekdays"`
}

type TimeRule struct {
	from     time.Time
	to       time.Time
	loc      *time.Location
	weekdays []time.Weekday
}

func (f *TimeRule) Prepare(
	_ wrapper.Entity,
	_ zerolog.Logger,
) error {
	return nil
}

func (f *TimeRule) Apply(
	_ wrapper.Entity,
	logger zerolog.Logger,
) (bool, error) {
	n := time.Now().In(f.loc)

	d := n.Weekday()
	if !slices.Contains(f.weekdays, d) {
		return false, nil
	}

	now, _ := time.ParseInLocation(
		"15:04",
		fmt.Sprintf("%02d:%02d", n.Hour(), n.Minute()),
		f.loc,
	)
	fromLtTo := f.from.Before(f.to) && (now.Before(f.from) || now.After(f.to))
	fromGtTo := f.from.After(f.to) && (now.Before(f.from) && now.After(f.to))
	if fromLtTo || fromGtTo {
		return false, nil
	}

	logger.Debug().Stringer("match", n).Msg("Time match")
	return true, nil
}

func (f *TimeRule) String() string {
	return fmt.Sprintf(
		"Time(from=%02d:%02d, to=%02d:%02d, weekdays=%s, timezone=%s)",
		f.from.Hour(),
		f.from.Minute(),
		f.to.Hour(),
		f.to.Minute(),
		common.FormatStringerSlice(f.weekdays),
		f.loc.String(),
	)
}

// NOTE: GeoParam and GeoRegexp must have same field names.
type GeoParam struct {
	Organisation []string `mapstructure:"organisation"`
	CountryCode  []string `mapstructure:"country_code"`
	Country      []string `mapstructure:"country"`
	RegionCode   []string `mapstructure:"region_code"`
	Region       []string `mapstructure:"region"`
	City         []string `mapstructure:"city"`
	Timezone     []string `mapstructure:"timezone"`
	ASN          []string `mapstructure:"asn"`
}

type GeoParams struct {
	Path         string     `mapstructure:"list"`
	Geolocations []GeoParam `mapstructure:"geolocations"`
}

type GeoRegexp struct {
	Organisation []*regexp.Regexp
	CountryCode  []*regexp.Regexp
	Country      []*regexp.Regexp
	RegionCode   []*regexp.Regexp
	Region       []*regexp.Regexp
	City         []*regexp.Regexp
	Timezone     []*regexp.Regexp
	ASN          []*regexp.Regexp
}

func (r *GeoRegexp) String() string {
	return fmt.Sprintf(
		"geo(organisation=%s, country_code=%s, country=%s, "+
			"region_code=%s, region=%s, city=%s, timezone=%s, asn=%s)",
		common.FormatStringerSlice(r.Organisation),
		common.FormatStringerSlice(r.CountryCode),
		common.FormatStringerSlice(r.Country),
		common.FormatStringerSlice(r.RegionCode),
		common.FormatStringerSlice(r.Region),
		common.FormatStringerSlice(r.City),
		common.FormatStringerSlice(r.Timezone),
		common.FormatStringerSlice(r.ASN),
	)
}

type GeoRule struct {
	db         *database.DB
	path       string
	list       []*regexp.Regexp
	geo        []*GeoRegexp
	apicounter *atomic.Int32
	ipapico    ipapico.Client
	ipapicom   ipapicom.Client
}

func (f *GeoRule) Prepare(
	e wrapper.Entity,
	logger zerolog.Logger,
) error {
	_, err := f.getGeoInfoByEntity(e, logger)
	if err != nil {
		return fmt.Errorf("can't prepare geolocation info: %w", err)
	}
	return nil
}

func (f *GeoRule) Apply(
	e wrapper.Entity,
	logger zerolog.Logger,
) (bool, error) {
	geo, err := f.getGeoInfoByEntity(e, logger)
	if err != nil {
		return false, fmt.Errorf("can't get geolocation info: %w", err)
	}

	if f.findByRegexp(geo, logger) {
		return true, nil
	}
	for _, gr := range f.geo {
		m := f.findByGeoRegexp(geo, gr)
		if m {
			logger.Debug().Stringer("match", gr).Msg("Geo match")
			return true, nil
		}
	}
	return false, nil
}

func (f *GeoRule) getGeoInfoByEntity(
	e wrapper.Entity,
	logger zerolog.Logger,
) (*database.Geolocation, error) {
	ip := e.GetIP().String()

	geo, err := f.db.GetGeolocation(ip)
	if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
		return nil, fmt.Errorf("can't get cached geolocation: %w", err)
	}
	if geo != nil {
		return geo, nil
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second*5, //nolint:gomnd
	)
	defer cancel()

	geo = &database.Geolocation{}
	switch f.apicounter.Inc() % 2 {
	case 0:
		var g *ipapico.Location
		g, err = f.ipapico.GetLocationForIP(ctx, ip)
		if err != nil && !errors.Is(err, ipapico.ErrReservedRange) {
			return nil, fmt.Errorf(
				"can't get geolocation with ipapi.co: %w",
				err,
			)
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
		g, err = f.ipapicom.GetLocationForIP(ctx, ip)
		if err != nil && !(errors.Is(err, ipapicom.ErrReservedRange) ||
			errors.Is(err, ipapicom.ErrPrivateRange)) {
			return nil, fmt.Errorf(
				"can't get geolocation with ip-api.com: %w",
				err,
			)
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

func (f *GeoRule) findByRegexp(
	geo *database.Geolocation,
	logger zerolog.Logger,
) bool {
	if len(f.list) == 0 {
		return false
	}

	// iterate fields and match "list" regexps on them
	gs := reflect.ValueOf(geo).Elem()
	for i := 0; i < gs.NumField(); i++ {
		gv := gs.Field(i)
		if gv.Len() == 0 {
			continue
		}

		for _, re := range f.list {
			switch v := gv.Interface().(type) {
			case []string:
				for _, s := range v {
					if re.MatchString(s) {
						logger.Debug().
							Stringer("match", re).
							Msg("Geo regexp match")
						return true
					}
				}
			case string:
				if re.MatchString(v) {
					logger.Debug().Stringer("match", re).Msg("Geo regexp match")
					return true
				}
			}
		}
	}
	return false
}

func (f *GeoRule) findByGeoRegexp(
	geo *database.Geolocation,
	gr *GeoRegexp,
) bool {
	var found bool
	// iterate field regexps and apply them on fields
	grs := reflect.ValueOf(gr).Elem()
	for i := 0; i < grs.NumField(); i++ {
		fn := grs.Type().Field(i).Name
		gv := reflect.ValueOf(geo).Elem().FieldByName(fn)
		regexps, _ := grs.FieldByName(fn).Interface().([]*regexp.Regexp)
		if gv.Len() == 0 || len(regexps) == 0 {
			continue
		}
		// find regexp match of field fn.
		var m bool
		for _, re := range regexps {
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
		// if field does not match, stop checking other
		// fields with that GeoRegexp arr element
		if !m {
			found = false
			break
		}
		found = true
	}
	return found
}

func (f *GeoRule) String() string {
	return fmt.Sprintf(
		"Geo(path=%s, geolocations=%s)",
		f.path,
		common.FormatStringerSlice(f.geo),
	)
}

type ReverseLookupParams struct {
	DNS  string `mapstructure:"dns"`
	Path string `mapstructure:"list"`
}

type ReverseLookupRule struct {
	db   *database.DB
	path string
	dns  netip.AddrPort
	list []*regexp.Regexp
}

func (f *ReverseLookupRule) Prepare(
	e wrapper.Entity,
	logger zerolog.Logger,
) error {
	_, err := f.getDomainByEntity(e, logger)
	if err != nil {
		return fmt.Errorf("can't prepare reverse lookup info: %w", err)
	}
	return nil
}

func (f *ReverseLookupRule) Apply(
	e wrapper.Entity,
	logger zerolog.Logger,
) (bool, error) {
	ptr, err := f.getDomainByEntity(e, logger)
	if err != nil {
		return false, fmt.Errorf("can't get reverse lookup info: %w", err)
	}

	for _, d := range ptr.Domains {
		for _, re := range f.list {
			if re.MatchString(d) {
				logger.Debug().
					Stringer("match", re).
					Msg("Reverse lookup regexp match")
				return true, nil
			}
		}
	}
	return false, nil
}

func (f *ReverseLookupRule) getDomainByEntity(
	e wrapper.Entity,
	logger zerolog.Logger,
) (*database.ReverseLookup, error) {
	ip := e.GetIP().String()

	rl, err := f.db.GetReverseLookup(ip)
	if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
		return nil, fmt.Errorf("can't get cached reverse lookup: %w", err)
	}
	if rl != nil {
		return rl, nil
	}

	rl = &database.ReverseLookup{}

	addr, _ := dns.ReverseAddr(ip)
	m := new(dns.Msg)
	m.SetQuestion(addr, dns.TypePTR)
	m.RecursionDesired = true

	c := new(dns.Client)
	r, _, err := c.Exchange(m, f.dns.String())
	if err != nil {
		return nil, fmt.Errorf("can't create PTR dns request: %w", err)
	}

	for _, a := range r.Answer {
		ptr, ok := a.(*dns.PTR)
		if !ok {
			log.Error().
				Str("response", a.String()).
				Msg("Unknown dns response")
			continue
		}

		rl.Domains = append(rl.Domains, ptr.Ptr[:len(ptr.Ptr)-1])
	}

	logger.Debug().
		Strs("ptr", rl.Domains).
		Msg("New reverse lookup")
	err = f.db.SaveReverseLookup(ip, rl)
	if err != nil {
		return nil, fmt.Errorf("can't save reverse lookup: %w", err)
	}

	return rl, nil
}

func (f *ReverseLookupRule) String() string {
	return fmt.Sprintf(
		"ReverseLookup(list=%s, dns=%s)",
		f.path,
		f.dns.String(),
	)
}
