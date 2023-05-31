package filters

import (
	"errors"
	"fmt"
	"strings"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/database"
	"github.com/D00Movenok/BounceBack/internal/wrapper"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
)

var (
	ErrInvalidFilterArgs = errors.New("invalid filter arguments")
)

func NewCompositeAndFilter(_ *database.DB, fs FilterSet, cfg common.FilterConfig) (Filter, error) {
	var params CompositeAndFilterParams
	err := mapstructure.Decode(cfg.Params, &params)
	if err != nil {
		return nil, fmt.Errorf("can't decode params: %w", err)
	}

	if len(params.Filters) < 2 { //nolint:gomnd
		return nil, ErrInvalidFilterArgs
	}

	f := CompositeAndFilter{filters: make([]Filter, 0, len(params.Filters))}
	for _, name := range params.Filters {
		filter, ok := fs.Get(name)
		if !ok {
			return nil, fmt.Errorf("invalid filter name: %s", name)
		}
		f.filters = append(f.filters, filter)
	}

	return f, nil
}

func NewCompositeOrFilter(_ *database.DB, fs FilterSet, cfg common.FilterConfig) (Filter, error) {
	var params CompositeOrFilterParams
	err := mapstructure.Decode(cfg.Params, &params)
	if err != nil {
		return nil, fmt.Errorf("can't decode params: %w", err)
	}

	if len(params.Filters) < 2 { //nolint:gomnd
		return nil, ErrInvalidFilterArgs
	}

	f := CompositeOrFilter{filters: make([]Filter, 0, len(params.Filters))}
	for _, name := range params.Filters {
		filter, ok := fs.Get(name)
		if !ok {
			return nil, fmt.Errorf("invalid filter name: %s", name)
		}
		f.filters = append(f.filters, filter)
	}

	return f, nil
}

func NewCompositeNotFilter(_ *database.DB, fs FilterSet, cfg common.FilterConfig) (Filter, error) {
	var params CompositeNotFilterParams
	err := mapstructure.Decode(cfg.Params, &params)
	if err != nil {
		return nil, fmt.Errorf("can't decode params: %w", err)
	}

	if params.Filter == "" {
		return nil, ErrInvalidFilterArgs
	}

	name := params.Filter
	f, ok := fs.Get(name)
	if !ok {
		return nil, fmt.Errorf("invalid filter name: %s", name)
	}

	return CompositeNotFilter{filter: f}, nil
}

type CompositeAndFilterParams struct {
	Filters []string `json:"filters" mapstructure:"filters"`
}

type CompositeAndFilter struct {
	filters []Filter
}

func (f CompositeAndFilter) Apply(e wrapper.Entity, logger zerolog.Logger) (bool, error) {
	for _, filter := range f.filters {
		res, err := filter.Apply(e, logger)
		if err != nil {
			return false, fmt.Errorf("error in filter %T: %w", filter, err)
		}
		if !res {
			return false, nil
		}
	}
	return true, nil
}

func (f CompositeAndFilter) String() string {
	filterNames := make([]string, 0, len(f.filters))
	for _, f := range f.filters {
		filterNames = append(filterNames, f.String())
	}
	return strings.Join(filterNames, " and ")
}

type CompositeOrFilterParams struct {
	Filters []string `json:"filters" mapstructure:"filters"`
}

type CompositeOrFilter struct {
	filters []Filter
}

func (r CompositeOrFilter) Apply(e wrapper.Entity, logger zerolog.Logger) (bool, error) {
	for _, filter := range r.filters {
		res, err := filter.Apply(e, logger)
		if err != nil {
			return false, fmt.Errorf("error in filter %T: %w", filter, err)
		}
		if res {
			return true, nil
		}
	}
	return false, nil
}

func (r CompositeOrFilter) String() string {
	filterNames := make([]string, 0, len(r.filters))
	for _, f := range r.filters {
		filterNames = append(filterNames, f.String())
	}
	return strings.Join(filterNames, " or ")
}

type CompositeNotFilterParams struct {
	Filter string `json:"filter" mapstructure:"filter"`
}

type CompositeNotFilter struct {
	filter Filter
}

func (f CompositeNotFilter) Apply(e wrapper.Entity, logger zerolog.Logger) (bool, error) {
	res, err := f.filter.Apply(e, logger)
	if err != nil {
		return false, fmt.Errorf("error in filter %T: %w", f.filter, err)
	}
	return !res, nil
}

func (f CompositeNotFilter) String() string {
	return fmt.Sprintf("not (%s)", f.filter)
}
