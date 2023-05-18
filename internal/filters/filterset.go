package filters

import (
	"fmt"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/rs/zerolog/log"
)

type FilterSet struct {
	Filters map[string]Filter
}

func (fs *FilterSet) Get(name string) (Filter, bool) {
	if filter, ok := fs.Filters[name]; ok {
		return filter, true
	}
	return nil, false
}

func NewFilterSet(cfg []common.FilterConfig) (*FilterSet, error) {
	fs := FilterSet{Filters: map[string]Filter{}}

	for _, fc := range cfg {
		newFilter, ok := GetDefaultFilters()[fc.Type]
		if !ok {
			return nil, fmt.Errorf("unknown type for filter \"%s\": %s", fc.Name, fc.Type)
		}

		filter, err := newFilter(fc)
		if err != nil {
			return nil, fmt.Errorf("can't create filter \"%s\": %w", fc.Name, err)
		}
		fs.Filters[fc.Name] = filter

		log.Debug().Str("name", fc.Name).Stringer("filter", filter).Msg("Created new filter")
	}

	return &fs, nil
}
