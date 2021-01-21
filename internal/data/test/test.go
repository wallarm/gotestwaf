package test

import (
	"sync"
)

type DB struct {
	mu          sync.Mutex
	counters    map[string]map[string]map[bool]int
	passedTests []Test
	failedTests []Test
	naTests     []Test
	tests       []TestCase
}

func NewDB(tests []TestCase) *DB {
	r := DB{
		counters: make(map[string]map[string]map[bool]int),
		tests:    tests,
	}

	for _, test := range tests {
		if _, ok := r.counters[test.Set]; !ok {
			r.counters[test.Set] = map[string]map[bool]int{}
		}
		if _, ok := r.counters[test.Set][test.Name]; !ok {
			r.counters[test.Set][test.Name] = map[bool]int{}
		}
	}
	return &r
}

func (db *DB) UpdatePassedTests(t *Test) {
	db.mu.Lock()
	db.counters[t.TestSet][t.TestCase][true]++
	db.passedTests = append(db.passedTests, *t)
	defer db.mu.Unlock()
}

func (db *DB) UpdateNaTests(t *Test, tpe bool) {
	db.mu.Lock()
	db.counters[t.TestSet][t.TestCase][tpe]++
	db.passedTests = append(db.passedTests, *t)
	defer db.mu.Unlock()
}

func (db *DB) UpdateFailedTests(t *Test) {
	db.mu.Lock()
	db.counters[t.TestSet][t.TestCase][false]++
	db.failedTests = append(db.failedTests, *t)
	defer db.mu.Unlock()
}

func (db *DB) GetTests() []TestCase {
	return db.tests
}
