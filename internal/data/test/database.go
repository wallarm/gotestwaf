package test

import (
	"sync"
)

type DB struct {
	mu           sync.Mutex
	counters     map[string]map[string]map[string]int
	passedTests  []Info
	blockedTests []Info
	failedTests  []Info
	naTests      []Info
	tests        []Case

	overallPassedRequestsPercentage float32
	overallCompletedTestCases       float32
	overallRequests                 int
	overallRequestsBlocked          int
	wafScore                        float32
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
	}
	return &r
}

func (db *DB) UpdatePassedTests(t *Info) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.counters[t.Set][t.Case]["passed"]++
	db.passedTests = append(db.passedTests, *t)
}

func (db *DB) UpdateNaTests(t *Info, nonBlockedAsPassed bool) {
	db.mu.Lock()
	defer db.mu.Unlock()
	if nonBlockedAsPassed {
		db.counters[t.Set][t.Case]["passed"]++
	} else {
		db.counters[t.Set][t.Case]["blocked"]++
	}
	db.naTests = append(db.naTests, *t)
}

func (db *DB) UpdateBlockedTests(t *Info) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.counters[t.Set][t.Case]["blocked"]++
	db.blockedTests = append(db.blockedTests, *t)
}

func (db *DB) UpdateFailedTests(t *Info) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.counters[t.Set][t.Case]["failed"]++
	db.failedTests = append(db.failedTests, *t)
}

func (db *DB) GetTestCases() []Case {
	return db.tests
}
