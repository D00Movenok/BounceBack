package filters

func GetDefaultFilterWrappers() map[string]FilterWrapperCreator {
	return map[string]FilterWrapperCreator{
		"not": NewNotWrapper,
	}
}

func GetDefaultFilterBase() map[string]FilterBaseCreator {
	return map[string]FilterBaseCreator{
		// boolean
		"and": NewCompositeAndFilter,
		"or":  NewCompositeOrFilter,
		"not": NewCompositeNotFilter,
		// ip filters
		"ip":  NewIPFilter,
		"geo": NewGeolocationFilter,
		// misc
		"time": NewTimeFilter,
	}
}
