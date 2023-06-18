package database

import (
	badger "github.com/dgraph-io/badger/v3"
)

type DB struct {
	DB *badger.DB
}

// Init open DB connection and run migrations.
func New(path string, inMemory bool) (*DB, error) {
	bc := badger.DefaultOptions(path).WithInMemory(inMemory)
	bc.Logger = nil
	d, err := badger.Open(bc)
	if err != nil {
		return nil, err
	}
	db := &DB{
		DB: d,
	}
	return db, err
}
