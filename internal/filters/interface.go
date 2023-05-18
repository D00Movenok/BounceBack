package filters

import (
	"fmt"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/wrapper"
)

type Filter interface {
	Apply(wrapper.Entity) (bool, error)
	fmt.Stringer
}

type FilterCreator func(common.FilterConfig) (Filter, error)
