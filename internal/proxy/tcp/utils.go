package tcp

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"
)

func isConnectionClosedErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrDropped) {
		return true
	}
	return strings.Contains(err.Error(), "use of closed network connection")
}

func parseSchemeAddrPort(url string) (string, netip.AddrPort, error) {
	split := strings.Split(url, "://")
	if len(split) != 2 { //nolint: gomnd // scheme + addrport
		return "", netip.AddrPort{}, &InvalidSchemeAddrPortError{url: url}
	}

	ap, err := netip.ParseAddrPort(split[1])
	if err != nil {
		return "", netip.AddrPort{}, fmt.Errorf(
			"can't parse AddrPort \"%s\": %w",
			split[1],
			err,
		)
	}

	return split[0], ap, nil
}
