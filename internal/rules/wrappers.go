package rules

import (
	"github.com/D00Movenok/BounceBack/internal/common"
)

func NewNotWrapper(r Rule, _ common.RuleConfig) Rule {
	return CompositeNotRule{rule: r}
}
