package rules

import (
	"errors"
	"fmt"
)

var (
	ErrEmptyRuleType   = errors.New("empty rule type")
	ErrInvalidRuleArgs = errors.New("invalid rule arguments")
	ErrOddOrZero       = errors.New("data length is odd or equal zero")
	ErrCaseMismatch    = errors.New("case mismatch")
)

type UnknownBaseRuleError struct {
	rule  string
	token string
}

func (e UnknownBaseRuleError) Error() string {
	return fmt.Sprintf("unknown rule type for \"%s\": %s", e.rule, e.token)
}

type UnknownWrapperRuleError struct {
	rule  string
	token string
}

func (e UnknownWrapperRuleError) Error() string {
	return fmt.Sprintf(
		"unknown rule wrapper for \"%s\": %s",
		e.rule,
		e.token,
	)
}

type InvalidRuleNameError struct {
	rule string
}

func (e InvalidRuleNameError) Error() string {
	return fmt.Sprintf("invalid rule name: %s", e.rule)
}

type UnknownDayOfWeekError struct {
	day string
}

func (e UnknownDayOfWeekError) Error() string {
	return fmt.Sprintf("unknown day of week: %s", e.day)
}

type UnknownTransformError struct {
	transform string
}

func (e UnknownTransformError) Error() string {
	return fmt.Sprintf("unknown transform: %s", e.transform)
}
