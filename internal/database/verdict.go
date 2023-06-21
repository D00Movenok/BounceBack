package database

import (
	"errors"
	"fmt"

	badger "github.com/dgraph-io/badger/v3"
)

const (
	VerdictPrefix string = "ip-verdict-"

	VerdictNone = iota
	VerdictAccept
	VerdictReject
)

type Verdict struct {
	Accepts uint
	Rejects uint
}

func (db *DB) GetVerdict(ip string) (*Verdict, error) {
	v, err := getCache[Verdict](db, ip, VerdictPrefix)
	if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
		return nil, fmt.Errorf("can't get cached verdicts: %w", err)
	}
	if v == nil {
		return &Verdict{}, nil
	}
	return v, nil
}

func (db *DB) IncAccepts(ip string) error {
	v, err := db.GetVerdict(ip)
	if err != nil {
		return fmt.Errorf("can't get verdicts for accepts inc: %w", err)
	}
	v.Accepts++
	return saveCache(db, ip, VerdictPrefix, v)
}

func (db *DB) IncRejects(ip string) error {
	v, err := db.GetVerdict(ip)
	if err != nil {
		return fmt.Errorf("can't get verdicts for rejects inc: %w", err)
	}
	v.Rejects++
	return saveCache(db, ip, VerdictPrefix, v)
}
