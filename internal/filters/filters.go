package filters

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
