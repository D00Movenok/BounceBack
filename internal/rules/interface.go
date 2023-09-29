package rules

import (
	"fmt"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/database"
	"github.com/D00Movenok/BounceBack/internal/wrapper"
	"github.com/rs/zerolog"
)

type Rule interface {
	Prepare(wrapper.Entity, zerolog.Logger) error
	Apply(wrapper.Entity, zerolog.Logger) (bool, error)
	fmt.Stringer
}

type RuleBaseCreator func(
	db *database.DB,
	rs RuleSet,
	cfg common.RuleConfig,
	globals common.Globals,
) (Rule, error)

type RuleWrapperCreator func(
	rule Rule,
	cfg common.RuleConfig,
) Rule
