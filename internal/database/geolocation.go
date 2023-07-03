package database

const GeolocationPrefix string = "ip-geo-"

type Geolocation struct {
	Organisation []string
	CountryCode  string
	Country      string
	RegionCode   string
	Region       string
	City         string
	Timezone     string
	ASN          string
}

func (db *DB) GetGeolocation(ip string) (*Geolocation, error) {
	return getCache[Geolocation](db, ip, GeolocationPrefix)
}

func (db *DB) SaveGeolocation(ip string, geo *Geolocation) error {
	return saveCache(db, ip, GeolocationPrefix, geo)
}
