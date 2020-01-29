package report

import "sync"

type Report struct {
	Report map[string]map[string]map[bool]int
	Lock   sync.RWMutex
}

func CreateReport() Report {
	r := Report{}
	r.Lock = sync.RWMutex{}
	r.Report = make(map[string]map[string]map[bool]int)
	return r
}
