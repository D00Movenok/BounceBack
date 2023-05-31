package filters

import (
	"fmt"
	"strings"
)

func FormatStringerSlice[T fmt.Stringer](s []T) string {
	slice := make([]string, 0, len(s))
	for _, d := range s {
		slice = append(slice, d.String())
	}
	return "[" + strings.Join(slice, ", ") + "]"
}
