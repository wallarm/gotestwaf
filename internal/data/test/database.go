package test

import (
	"sync"
)

type DB struct {
	mu          sync.Mutex
	counters    map[string]map[string]map[bool]int
	passedTests []Info
	failedTests []Info
	naTests     []Info
	tests       []Case

	overallPassedRate         float32
	overallTestcasesCompleted float32
	overallTestsCompleted     int
	overallTestsFailed        int
	wafScore                  float32
}

func NewDB(tests []Case) *DB {
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

func (db *DB) UpdatePassedTests(t *Info) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.counters[t.Set][t.Case][true]++
	db.passedTests = append(db.passedTests, *t)
}

func (db *DB) UpdateNaTests(t *Info, nonBlockedAsPassed bool) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.counters[t.Set][t.Case][nonBlockedAsPassed]++
	db.naTests = append(db.naTests, *t)
}

func (db *DB) UpdateFailedTests(t *Info) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.counters[t.Set][t.Case][false]++
	db.failedTests = append(db.failedTests, *t)
}

func (db *DB) GetTestCases() []Case {
	return db.tests
}
