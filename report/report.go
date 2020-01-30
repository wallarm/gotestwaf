package report

import "sync"

type ReportKey struct {
	Testset, Name string
}

type Report struct {
	Report map[ReportKey]map[bool]int
	Lock   *sync.RWMutex
	Wg     *sync.WaitGroup
}

func CreateReport() *Report {
	return &Report{
		Report: make(map[ReportKey]map[bool]int),
		Lock:   &sync.RWMutex{},
		Wg:     &sync.WaitGroup{},
	}
}
