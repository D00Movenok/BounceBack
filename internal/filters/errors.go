package filters

import (
	"errors"
	"fmt"
)

var (
	ErrEmptyFilterType   = errors.New("empty filter type")
	ErrInvalidFilterArgs = errors.New("invalid filter arguments")
	ErrOddOrZero         = errors.New("data length is odd or equal zero")
	ErrCaseMismatch      = errors.New("case mismatch")
)

type UnknownBaseFilterError struct {
	filter string
	token  string
}

func (e UnknownBaseFilterError) Error() string {
	return fmt.Sprintf("unknown filter type for \"%s\": %s", e.filter, e.token)
}

type UnknownWrapperFilterError struct {
	filter string
	token  string
}

func (e UnknownWrapperFilterError) Error() string {
	return fmt.Sprintf(
		"unknown filter wrapper for \"%s\": %s",
		e.filter,
		e.token,
	)
}

type InvalidFilterNameError struct {
	filter string
}

func (e InvalidFilterNameError) Error() string {
	return fmt.Sprintf("invalid filter name: %s", e.filter)
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
