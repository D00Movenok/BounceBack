package database

import (
	"bytes"
	"fmt"

	xdr "github.com/davecgh/go-xdr/xdr2"
	badger "github.com/dgraph-io/badger/v3"
)

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
	var geo *Geolocation
	err := db.DB.View(func(txn *badger.Txn) error {
		v, err := txn.Get([]byte(GeolocationPrefix + ip))
		if err != nil {
			return fmt.Errorf("can't get value from storage: %w", err)
		}
		b, err := v.ValueCopy(nil)
		if err != nil {
			return fmt.Errorf("can't copy value: %w", err)
		}
		_, err = xdr.Unmarshal(bytes.NewReader(b), &geo)
		if err != nil {
			return fmt.Errorf("can't unmarshal value: %w", err)
		}
		return nil
	})
	return geo, err
}

func (db *DB) SaveGeolocation(ip string, geo *Geolocation) error {
	err := db.DB.Update(func(txn *badger.Txn) error {
		var w bytes.Buffer
		_, err := xdr.Marshal(&w, geo)
		if err != nil {
			return fmt.Errorf("can't marshal value: %w", err)
		}
		err = txn.Set([]byte(GeolocationPrefix+ip), w.Bytes())
		if err != nil {
			return fmt.Errorf("can't save value to storage: %w", err)
		}
		return nil
	})
	return err
}
