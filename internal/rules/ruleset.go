package rules

import (
	"fmt"
	"strings"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/database"
	"github.com/rs/zerolog/log"
)

type RuleSet struct {
	Rules map[string]Rule
}

func (rs *RuleSet) Get(name string) (Rule, bool) {
	if rule, ok := rs.Rules[name]; ok {
		return rule, true
	}
	return nil, false
}

func NewRuleSet(
	db *database.DB,
	cfg []common.RuleConfig,
	globals common.Globals,
) (*RuleSet, error) {
	rs := RuleSet{Rules: map[string]Rule{}}

	for _, rc := range cfg {
		tokens := strings.Split(rc.Type, "::")
		if len(tokens) == 0 {
			return nil, ErrEmptyRuleType
		}

		var (
			err  error
			rule Rule
		)

		lastToken := tokens[len(tokens)-1]
		if newRule, ok := GetRuleBase()[lastToken]; ok {
			if rule, err = newRule(db, rs, rc, globals); err != nil {
				return nil, fmt.Errorf(
					"can't create base rule for \"%s\": %w",
					rc.Name,
					err,
				)
			}
		} else {
			return nil, &UnknownBaseRuleError{
				rule:  rc.Name,
				token: lastToken,
			}
		}

		// iterate tokens without last
		for i := len(tokens) - 2; i >= 0; i-- { //nolint:gomnd
			wrapperName := tokens[i]
			if wrapperCreator, ok := GetRuleWrappers()[wrapperName]; ok {
				rule = wrapperCreator(rule, rc)
			} else {
				return nil, &UnknownBaseRuleError{
					rule:  rc.Name,
					token: wrapperName,
				}
			}
		}

		rs.Rules[rc.Name] = rule

		log.Debug().
			Str("name", rc.Name).
			Stringer("rule", rule).
			Msg("Created new rule")
	}

	return &rs, nil
}
