package db

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"sync"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/payload/placeholder"
)

type DB struct {
	sync.Mutex

	counters     map[string]map[string]map[string]int
	passedTests  []*Info
	blockedTests []*Info
	failedTests  []*Info
	naTests      []*Info
	tests        []*Case

	scannedPaths map[string]map[string]interface{}

	NumberOfTests uint
	Hash          string

	IsGrpcAvailable bool
}

func NewDB(tests []*Case) (*DB, error) {
	db := &DB{
		counters: make(map[string]map[string]map[string]int),
		tests:    tests,
	}

	var encodedCase bytes.Buffer

	enc := gob.NewEncoder(&encodedCase)
	gob.Register(placeholder.RawRequestConfig{})
	sha256hash := sha256.New()

	for _, test := range tests {
		if _, ok := db.counters[test.Set]; !ok {
			db.counters[test.Set] = map[string]map[string]int{}
		}
		if _, ok := db.counters[test.Set][test.Name]; !ok {
			db.counters[test.Set][test.Name] = map[string]int{}
		}

		db.NumberOfTests += uint(len(test.Payloads) * len(test.Encoders) * len(test.Placeholders))

		err := enc.Encode(*test)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't encode test case")
		}

		sha256hash.Write(encodedCase.Bytes())
		encodedCase.Reset()
	}

	db.Hash = hex.EncodeToString(sha256hash.Sum(nil)[:16])

	return db, nil
}

func (db *DB) UpdatePassedTests(t *Info) {
	db.Lock()
	defer db.Unlock()
	db.counters[t.Set][t.Case]["passed"]++
	db.passedTests = append(db.passedTests, t)
}

func (db *DB) UpdateNaTests(t *Info, ignoreUnresolved, nonBlockedAsPassed, isTruePositive bool) {
	db.Lock()
	defer db.Unlock()
	if (ignoreUnresolved || nonBlockedAsPassed) && isTruePositive {
		db.counters[t.Set][t.Case]["passed"]++
	} else {
		db.counters[t.Set][t.Case]["blocked"]++
	}
	db.naTests = append(db.naTests, t)
}

func (db *DB) UpdateBlockedTests(t *Info) {
	db.Lock()
	defer db.Unlock()
	db.counters[t.Set][t.Case]["blocked"]++
	db.blockedTests = append(db.blockedTests, t)
}

func (db *DB) UpdateFailedTests(t *Info) {
	db.Lock()
	defer db.Unlock()
	db.counters[t.Set][t.Case]["failed"]++
	db.failedTests = append(db.failedTests, t)
}

func (db *DB) AddToScannedPaths(method string, path string) {
	db.Lock()
	defer db.Unlock()

	if db.scannedPaths == nil {
		db.scannedPaths = make(map[string]map[string]interface{})
	}

	if _, ok := db.scannedPaths[path]; !ok {
		db.scannedPaths[path] = make(map[string]interface{})
	}
	db.scannedPaths[path][method] = nil
}

func (db *DB) GetTestCases() []*Case {
	return db.tests
}
