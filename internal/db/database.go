package db

import (
	"sync"
)

type DB struct {
	sync.Mutex

	counters     map[string]map[string]map[string]int
	passedTests  []Info
	blockedTests []Info
	failedTests  []Info
	naTests      []Info
	tests        []Case

	numberOfTests uint
}

func NewDB(tests []Case) *DB {
	r := DB{
		counters: make(map[string]map[string]map[string]int),
		tests:    tests,
	}

	for _, test := range tests {
		if _, ok := r.counters[test.Set]; !ok {
			r.counters[test.Set] = map[string]map[string]int{}
		}
		if _, ok := r.counters[test.Set][test.Name]; !ok {
			r.counters[test.Set][test.Name] = map[string]int{}
		}

		r.numberOfTests += uint(len(test.Payloads) * len(test.Encoders) * len(test.Placeholders))
	}

	return &r
}

func (db *DB) UpdatePassedTests(t *Info) {
	db.Lock()
	defer db.Unlock()
	db.counters[t.Set][t.Case]["passed"]++
	db.passedTests = append(db.passedTests, *t)
}

func (db *DB) UpdateNaTests(t *Info, ignoreUnresolved, nonBlockedAsPassed, isTruePositive bool) {
	db.Lock()
	defer db.Unlock()
	if (ignoreUnresolved || nonBlockedAsPassed) && isTruePositive {
		db.counters[t.Set][t.Case]["passed"]++
	} else {
		db.counters[t.Set][t.Case]["blocked"]++
	}
	db.naTests = append(db.naTests, *t)
}

func (db *DB) UpdateBlockedTests(t *Info) {
	db.Lock()
	defer db.Unlock()
	db.counters[t.Set][t.Case]["blocked"]++
	db.blockedTests = append(db.blockedTests, *t)
}

func (db *DB) UpdateFailedTests(t *Info) {
	db.Lock()
	defer db.Unlock()
	db.counters[t.Set][t.Case]["failed"]++
	db.failedTests = append(db.failedTests, *t)
}

func (db *DB) GetTestCases() []Case {
	return db.tests
}

func (db *DB) GetNumberOfAllTestCases() uint {
	return db.numberOfTests
}
