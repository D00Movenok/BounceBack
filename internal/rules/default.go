package rules

func GetRuleWrappers() map[string]RuleWrapperCreator {
	return map[string]RuleWrapperCreator{
		"not": NewNotWrapper,
	}
}

func GetRuleBase() map[string]RuleBaseCreator {
	return map[string]RuleBaseCreator{
		// boolean
		"and": NewCompositeAndRule,
		"or":  NewCompositeOrRule,
		"not": NewCompositeNotRule,
		// ip rules
		"ip":             NewIPRule,
		"geo":            NewGeolocationRule,
		"reverse_lookup": NewReverseLookupRule,
		// packet inspection
		"regexp":    NewRegexpRule,
		"malleable": NewMalleableRule,
		// misc
		"time": NewTimeRule,
	}
}
