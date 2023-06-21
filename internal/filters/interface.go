package filters

import (
	"fmt"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/database"
	"github.com/D00Movenok/BounceBack/internal/wrapper"
	"github.com/rs/zerolog"
)

type Filter interface {
	Apply(wrapper.Entity, zerolog.Logger) (bool, error)
	fmt.Stringer
}

type FilterBaseCreator func(
	db *database.DB,
	fs FilterSet,
	cfg common.FilterConfig,
) (Filter, error)

type FilterWrapperCreator func(
	filter Filter,
	cfg common.FilterConfig,
) Filter
