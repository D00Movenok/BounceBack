package filters

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"time"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/wrapper"
	"github.com/mitchellh/mapstructure"
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
		return nil, fmt.Errorf("can't open banlist file: %w", err)
	}

	s := bufio.NewScanner(file)
	for s.Scan() {
		line := s.Bytes()
		line, _, _ = bytes.Cut(line, []byte("#")) // remove comment
		line, _, _ = bytes.Cut(line, []byte(" ")) // remove space after
		strLine := string(line)
		isSubnet := bytes.Contains(line, []byte{'/'})
		if isSubnet {
			subnet, err = netip.ParsePrefix(strLine)
			filter.subnetBanlist = append(filter.subnetBanlist, subnet)
		} else if strLine != "" {
			ip, err = netip.ParseAddr(strLine)
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
		return nil, fmt.Errorf("can't parse from time: %w", err)
	}
	to, err := time.ParseInLocation("15:04", params.To, loc)
	if err != nil {
		return nil, fmt.Errorf("can't parse to time: %w", err)
	}

	filter := &WorkTimeFilter{
		from: from,
		to:   to,
		loc:  loc,
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
	for _, s := range f.subnetBanlist {
		ok := s.Contains(ip)
		if ok {
			return true, nil
		}
	}
	for _, i := range f.ipBanlist {
		ok := i.Compare(ip) == 0
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func (f *IPFilter) String() string {
	return fmt.Sprintf("IP(list=%s)", f.path)
}

type WorkTimeParams struct {
	From     string `json:"from" mapstructure:"from"`
	To       string `json:"to" mapstructure:"to"`
	Location string `json:"timezone" mapstructure:"timezone"`
}

type WorkTimeFilter struct {
	from time.Time
	to   time.Time
	loc  *time.Location
}

func (f *WorkTimeFilter) Apply(_ wrapper.Entity) (bool, error) {
	n := time.Now().In(f.loc)
	now, _ := time.ParseInLocation("15:04", fmt.Sprintf("%02d:%02d", n.Hour(), n.Minute()), f.loc)
	fromLtTo := f.from.Before(f.to) && (now.Before(f.from) || now.After(f.to))
	fromGtTo := f.from.After(f.to) && (now.Before(f.from) && now.After(f.to))
	if fromLtTo || fromGtTo {
		return true, nil
	}
	return false, nil
}

func (f *WorkTimeFilter) String() string {
	return fmt.Sprintf("WorkTime(from=%02d:%02d, to=%02d:%02d, timezone=%s)",
		f.from.Hour(), f.from.Minute(), f.to.Hour(), f.to.Minute(), f.loc.String())
}
