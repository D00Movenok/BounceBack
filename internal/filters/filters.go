package filters

import (
	"bufio"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/wrapper"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

var (
	ErrInvalidFilterArgs = errors.New("invalid filter arguments")
)

func NewIPFilter(_ FilterSet, cfg common.FilterConfig) (Filter, error) {
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

func NewWorkTimeFilter(_ FilterSet, cfg common.FilterConfig) (Filter, error) {
	var params WorkTimeParams
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

	filter := &WorkTimeFilter{
		from:     from,
		to:       to,
		loc:      loc,
		weekdays: days,
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

func (f *IPFilter) Apply(e wrapper.Entity) (bool, error) {
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

type WorkTimeParams struct {
	From     string   `json:"from" mapstructure:"from"`
	To       string   `json:"to" mapstructure:"to"`
	Location string   `json:"timezone" mapstructure:"timezone"`
	Weekdays []string `json:"weekdays" mapstructure:"weekdays"`
}

type WorkTimeFilter struct {
	from     time.Time
	to       time.Time
	loc      *time.Location
	weekdays []time.Weekday
}

func (f *WorkTimeFilter) Apply(_ wrapper.Entity) (bool, error) {
	n := time.Now().In(f.loc)

	d := n.Weekday()
	if !slices.Contains(f.weekdays, d) {
		return true, nil
	}

	now, _ := time.ParseInLocation("15:04", fmt.Sprintf("%02d:%02d", n.Hour(), n.Minute()), f.loc)
	fromLtTo := f.from.Before(f.to) && (now.Before(f.from) || now.After(f.to))
	fromGtTo := f.from.After(f.to) && (now.Before(f.from) && now.After(f.to))
	if fromLtTo || fromGtTo {
		return true, nil
	}
	return false, nil
}

func (f *WorkTimeFilter) String() string {
	weekdaysNames := make([]string, 0, len(f.weekdays))
	for _, d := range f.weekdays {
		weekdaysNames = append(weekdaysNames, d.String())
	}
	return fmt.Sprintf("WorkTime(from=%02d:%02d, to=%02d:%02d, weekdays=%s, timezone=%s)",
		f.from.Hour(), f.from.Minute(), f.to.Hour(), f.to.Minute(), "["+strings.Join(weekdaysNames, ", ")+"]", f.loc.String())
}
