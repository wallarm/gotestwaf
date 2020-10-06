package report

import "sync"

type Test struct {
	Payload     string
	Encoder     string
	Placeholder string
	Testset     string
	Testcase    string
	StatusCode  int
}

type Report struct {
	Report      map[string]map[string]map[bool]int
	FailedTests []Test
	NaTests     []Test
	Lock        sync.RWMutex
}

func CreateReport() Report {
	r := Report{}
	r.Lock = sync.RWMutex{}
	r.Report = make(map[string]map[string]map[bool]int)
	r.FailedTests = make([]Test, 0)
	r.NaTests = make([]Test, 0)
	return r
}
