package base

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"

	"github.com/D00Movenok/BounceBack/internal/common"
	"golang.org/x/exp/slices"
)

type ActionNotAllowedError struct {
	action  string
	allowed []string
}

func (e ActionNotAllowedError) Error() string {
	return fmt.Sprintf(
		"action \"%s\" is not allowed, allowed actions: %s",
		e.action,
		common.FormatStringSlice(e.allowed),
	)
}

func verifyAction(a string, allowed []string) error {
	if !slices.Contains(allowed, a) {
		return &ActionNotAllowedError{action: a, allowed: allowed}
	}
	return nil
}

func IsConnectionClosed(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrDropped) {
		return true
	}
	return strings.Contains(err.Error(), "use of closed network connection")
}

func NetAddrToNetipAddrPort(a net.Addr) netip.AddrPort {
	return netip.MustParseAddrPort(a.String())
}
