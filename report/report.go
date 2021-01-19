package report

import "sync"

type Test struct {
	Payload     string
	Encoder     string
	Placeholder string
	TestSet     string
	TestCase    string
	StatusCode  int
}

type Report struct {
	Report      map[string]map[string]map[bool]int
	PassedTests []Test
	FailedTests []Test
	NaTests     []Test
	Mu          sync.RWMutex
}

func New() *Report {
	r := Report{}
	r.Mu = sync.RWMutex{}
	r.Report = make(map[string]map[string]map[bool]int)
	r.FailedTests = make([]Test, 0)
	r.NaTests = make([]Test, 0)
	r.PassedTests = make([]Test, 0)
	return &r
}
