package database

const ReverseLookupPrefix string = "ip-lookup-"

type ReverseLookup struct {
	Domains []string `json:"domains"`
}

func (db *DB) GetReverseLookup(ip string) (*ReverseLookup, error) {
	return getCache[ReverseLookup](db, ip, ReverseLookupPrefix)
}

func (db *DB) SaveReverseLookup(ip string, rl *ReverseLookup) error {
	return saveCache(db, ip, ReverseLookupPrefix, rl)
}
