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

type FilterBaseCreator func(fs FilterSet, cfg common.FilterConfig) (Filter, error)
type FilterWrapperCreator func(filter Filter, cfg common.FilterConfig) Filter
