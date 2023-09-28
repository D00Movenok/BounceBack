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

func NewFilterSet(
	db *database.DB,
	cfg []common.FilterConfig,
	globals common.Globals,
) (*FilterSet, error) {
	fs := FilterSet{Filters: map[string]Filter{}}

	for _, fc := range cfg {
		tokens := strings.Split(fc.Type, "::")
		if len(tokens) == 0 {
			return nil, ErrEmptyFilterType
		}

		var (
			err    error
			filter Filter
		)

		lastToken := tokens[len(tokens)-1]
		if newFilter, ok := GetFilterBase()[lastToken]; ok {
			if filter, err = newFilter(db, fs, fc, globals); err != nil {
				return nil, fmt.Errorf(
					"can't create base filter for \"%s\": %w",
					fc.Name,
					err,
				)
			}
		} else {
			return nil, &UnknownBaseFilterError{
				filter: fc.Name,
				token:  lastToken,
			}
		}

		// iterate tokens without last
		for i := len(tokens) - 2; i >= 0; i-- { //nolint:gomnd
			wrapperName := tokens[i]
			if wrapperCreator, ok := GetFilterWrappers()[wrapperName]; ok {
				filter = wrapperCreator(filter, fc)
			} else {
				return nil, &UnknownBaseFilterError{
					filter: fc.Name,
					token:  wrapperName,
				}
			}
		}

		fs.Filters[fc.Name] = filter

		log.Debug().
			Str("filter", fc.Name).
			Stringer("rule", filter).
			Msg("Created new filter")
	}

	return &fs, nil
}
