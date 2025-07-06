package rules

import (
	"fmt"
	"strings"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/database"
	"github.com/D00Movenok/BounceBack/internal/wrapper"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
)

func NewCompositeAndRule(
	_ *database.DB,
	rs RuleSet,
	cfg common.RuleConfig,
	_ common.Globals,
) (Rule, error) {
	var params CompositeAndRuleParams
	err := mapstructure.Decode(cfg.Params, &params)
	if err != nil {
		return nil, fmt.Errorf("can't decode params: %w", err)
	}

	if len(params.Rules) < 2 { //nolint:mnd
		return nil, ErrInvalidRuleArgs
	}

	f := CompositeAndRule{rules: make([]Rule, 0, len(params.Rules))}
	for _, name := range params.Rules {
		rule, ok := rs.Get(name)
		if !ok {
			return nil, &InvalidRuleNameError{rule: name}
		}
		f.rules = append(f.rules, rule)
	}

	return f, nil
}

func NewCompositeOrRule(
	_ *database.DB,
	rs RuleSet,
	cfg common.RuleConfig,
	_ common.Globals,
) (Rule, error) {
	var params CompositeOrRuleParams
	err := mapstructure.Decode(cfg.Params, &params)
	if err != nil {
		return nil, fmt.Errorf("can't decode params: %w", err)
	}

	if len(params.Rules) < 2 { //nolint:mnd
		return nil, ErrInvalidRuleArgs
	}

	f := CompositeOrRule{rules: make([]Rule, 0, len(params.Rules))}
	for _, name := range params.Rules {
		rule, ok := rs.Get(name)
		if !ok {
			return nil, &InvalidRuleNameError{rule: name}
		}
		f.rules = append(f.rules, rule)
	}

	return f, nil
}

func NewCompositeNotRule(
	_ *database.DB,
	rs RuleSet,
	cfg common.RuleConfig,
	_ common.Globals,
) (Rule, error) {
	var params CompositeNotRuleParams
	err := mapstructure.Decode(cfg.Params, &params)
	if err != nil {
		return nil, fmt.Errorf("can't decode params: %w", err)
	}

	if params.Rule == "" {
		return nil, ErrInvalidRuleArgs
	}

	name := params.Rule
	f, ok := rs.Get(name)
	if !ok {
		return nil, &InvalidRuleNameError{rule: name}
	}

	return CompositeNotRule{rule: f}, nil
}

type CompositeAndRuleParams struct {
	Rules []string `mapstructure:"rules"`
}

type CompositeAndRule struct {
	rules []Rule
}

func (f CompositeAndRule) Prepare(
	e wrapper.Entity,
	logger zerolog.Logger,
) error {
	return PrepareMany(
		f.rules,
		e,
		logger,
	)
}

func (f CompositeAndRule) Apply(
	e wrapper.Entity,
	logger zerolog.Logger,
) (bool, error) {
	for _, rule := range f.rules {
		res, err := rule.Apply(e, logger)
		if err != nil {
			return false, fmt.Errorf("error in rule \"%T\": %w", rule, err)
		}
		if !res {
			return false, nil
		}
	}
	return true, nil
}

func (f CompositeAndRule) String() string {
	ruleNames := make([]string, 0, len(f.rules))
	for _, f := range f.rules {
		ruleNames = append(ruleNames, f.String())
	}
	return strings.Join(ruleNames, " and ")
}

type CompositeOrRuleParams struct {
	Rules []string `mapstructure:"rules"`
}

type CompositeOrRule struct {
	rules []Rule
}

func (f CompositeOrRule) Prepare(
	e wrapper.Entity,
	logger zerolog.Logger,
) error {
	err := PrepareMany(f.rules, e, logger)
	if err != nil {
		return fmt.Errorf("can't prepare rules: %w", err)
	}
	return nil
}

func (f CompositeOrRule) Apply(
	e wrapper.Entity,
	logger zerolog.Logger,
) (bool, error) {
	for _, rule := range f.rules {
		res, err := rule.Apply(e, logger)
		if err != nil {
			return false, fmt.Errorf("error in rule \"%T\": %w", rule, err)
		}
		if res {
			return true, nil
		}
	}
	return false, nil
}

func (f CompositeOrRule) String() string {
	ruleNames := make([]string, 0, len(f.rules))
	for _, rule := range f.rules {
		ruleNames = append(ruleNames, rule.String())
	}
	return strings.Join(ruleNames, " or ")
}

type CompositeNotRuleParams struct {
	Rule string `mapstructure:"rule"`
}

type CompositeNotRule struct {
	rule Rule
}

func (f CompositeNotRule) Prepare(
	e wrapper.Entity,
	logger zerolog.Logger,
) error {
	return f.rule.Prepare(e, logger) //nolint: wrapcheck // gets from config
}

func (f CompositeNotRule) Apply(
	e wrapper.Entity,
	logger zerolog.Logger,
) (bool, error) {
	res, err := f.rule.Apply(e, logger)
	if err != nil {
		return false, fmt.Errorf("error in rule \"%T\": %w", f.rule, err)
	}
	return !res, nil
}

func (f CompositeNotRule) String() string {
	return fmt.Sprintf("not (%s)", f.rule)
}
