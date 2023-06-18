package filters

import (
	"github.com/D00Movenok/BounceBack/internal/common"
)

func NewNotWrapper(r Filter, _ common.FilterConfig) Filter {
	return CompositeNotFilter{filter: r}
}
