package filters

import (
	"bufio"
	"bytes"
	"fmt"
	"net/netip"
	"os"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/wrapper"
	"github.com/mitchellh/mapstructure"
)

func NewIPFilter(cfg common.FilterConfig) (Filter, error) {
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

type IPFilterParams struct {
	Path string `json:"banlist" mapstructure:"banlist"`
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
	return fmt.Sprintf("IPFilter(path=%s, ips=%d, subnets=%d)", f.path, len(f.ipBanlist), len(f.subnetBanlist))
}
