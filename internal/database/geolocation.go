package database

const GeolocationPrefix string = "ip-geo-"

type Geolocation struct {
	Organisation []string `json:"organisation"`
	CountryCode  string   `json:"country_code"`
	Country      string   `json:"country"`
	RegionCode   string   `json:"region_code"`
	Region       string   `json:"region"`
	City         string   `json:"city"`
	Timezone     string   `json:"timezone"`
	ASN          string   `json:"asn"`
}

func (db *DB) GetGeolocation(ip string) (*Geolocation, error) {
	return getCache[Geolocation](db, ip, GeolocationPrefix)
}

func (db *DB) SaveGeolocation(ip string, geo *Geolocation) error {
	return saveCache(db, ip, GeolocationPrefix, geo)
}
