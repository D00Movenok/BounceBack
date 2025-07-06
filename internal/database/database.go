package database

import (
	"bytes"
	"fmt"

	xdr "github.com/davecgh/go-xdr/xdr2"
	badger "github.com/dgraph-io/badger/v3"
)

type DB struct {
	DB *badger.DB
}

// New initializes open DB connection and run migrations.
func New(path string, inMemory bool) (*DB, error) {
	bc := badger.DefaultOptions(path).WithInMemory(inMemory)
	bc.Logger = nil
	d, err := badger.Open(bc)
	if err != nil {
		return nil, fmt.Errorf("can't open database: %w", err)
	}
	db := &DB{
		DB: d,
	}
	return db, nil
}

// get cache from cache db from "prefix-key".
// it is not a method, because go can't create generic methods.
func getCache[T any](db *DB, key string, prefix string) (*T, error) {
	var data *T
	err := db.DB.View(func(txn *badger.Txn) error {
		v, err := txn.Get([]byte(prefix + key))
		if err != nil {
			return fmt.Errorf("can't get value from storage: %w", err)
		}
		b, err := v.ValueCopy(nil)
		if err != nil {
			return fmt.Errorf("can't copy value: %w", err)
		}
		_, err = xdr.Unmarshal(bytes.NewReader(b), &data)
		if err != nil {
			return fmt.Errorf("can't unmarshal value: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("can't get cache: %w", err)
	}
	return data, nil
}

// save cache to cache db to "prefix-key".
// it is not a method, because of previous func.
func saveCache(db *DB, key string, prefix string, data any) error {
	err := db.DB.Update(func(txn *badger.Txn) error {
		var w bytes.Buffer
		_, err := xdr.Marshal(&w, data)
		if err != nil {
			return fmt.Errorf("can't marshal value: %w", err)
		}
		err = txn.Set([]byte(prefix+key), w.Bytes())
		if err != nil {
			return fmt.Errorf("can't save value to storage: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("can't save cache: %w", err)
	}
	return nil
}
