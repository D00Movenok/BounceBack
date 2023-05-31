package filters

import (
	"fmt"
	"strings"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/database"
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

func NewFilterSet(db *database.DB, cfg []common.FilterConfig) (*FilterSet, error) {
	fs := FilterSet{Filters: map[string]Filter{}}

	for _, fc := range cfg {
		tokens := strings.Split(fc.Type, "::")
		if len(tokens) == 0 {
			return nil, fmt.Errorf("invalid filter: %s", fc.Type)
		}

		var (
			err    error
			filter Filter
		)

		lastToken := tokens[len(tokens)-1]
		if newFilter, ok := GetDefaultFilterBase()[lastToken]; ok {
			if filter, err = newFilter(db, fs, fc); err != nil {
				return nil, fmt.Errorf("creating filter %s: %w", fc.Name, err)
			}
		} else {
			return nil, fmt.Errorf("invalid filter %s: last token invalid", fc.Type)
		}

		// iterate tokens without last
		for i := len(tokens) - 2; i >= 0; i-- { //nolint:gomnd
			wrapperName := tokens[i]
			if wrapperCreator, ok := GetDefaultFilterWrappers()[wrapperName]; ok {
				filter = wrapperCreator(filter, fc)
			} else {
				return nil, fmt.Errorf("unexpected token %s for %s", wrapperName, fc.Name)
			}
		}

		fs.Filters[fc.Name] = filter

		log.Debug().Str("name", fc.Name).Stringer("filter", filter).Msg("Created new filter")
	}

	return &fs, nil
}
