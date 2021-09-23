package config

import (
	"fmt"
	"sync"

	"github.com/wallarm/gotestwaf/internal/data/test"
	"github.com/wallarm/gotestwaf/internal/payload/encoder"
	"github.com/wallarm/gotestwaf/internal/payload/placeholder"
)

const Host = "localhost"
const Port = "8080"
const Address = Host + ":" + Port

type TestCasesMap struct {
	sync.Mutex
	m map[string]struct{}
}

func (tcm *TestCasesMap) CheckTestCaseAvailability(testCase string) bool {
	tcm.Lock()
	defer tcm.Unlock()
	if _, ok := tcm.m[testCase]; ok {
		delete(tcm.m, testCase)
		return true
	}
	return false
}

func (tcm *TestCasesMap) CountTestCases() int {
	tcm.Lock()
	defer tcm.Unlock()
	return len(tcm.m)
}

func GenerateTestCases() (testCases []test.Case, testCasesMap *TestCasesMap) {
	var encoders []string
	var placeholders []string
	testCasesMap = new(TestCasesMap)
	testCasesMap.m = make(map[string]struct{})

	for encoderName, _ := range encoder.Encoders {
		if encoderName == encoder.DefaultGRPCEncoder.GetName() {
			continue
		}
		encoders = append(encoders, encoderName)
	}

	for placeholderName := range placeholder.Placeholders {
		placeholders = append(placeholders, placeholderName)
	}

	payloads := []string{"bypassed", "blocked", "unresolved"}

	for _, ph := range placeholders {
		for _, enc := range encoders {
			testCases = append(testCases, test.Case{
				Payloads:       payloads,
				Encoders:       []string{enc},
				Placeholders:   []string{ph},
				Set:            "test-set",
				Name:           fmt.Sprintf("%s-%s", ph, enc),
				IsTruePositive: true,
			})
			for _, p := range payloads {
				testCasesMap.m[fmt.Sprintf("%s-%s-%s", p, ph, enc)] = struct{}{}
			}
		}
	}

	return
}
