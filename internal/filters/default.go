package filters

func GetDefaultFilters() map[string]FilterCreator {
	return map[string]FilterCreator{
		"ip_filter": NewIPFilter,
	}
}
