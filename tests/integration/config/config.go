package config

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"sync"

	"github.com/wallarm/gotestwaf/internal/data/config"
	"github.com/wallarm/gotestwaf/internal/data/test"
	"github.com/wallarm/gotestwaf/internal/payload/encoder"
)

const (
	Host        = "localhost"
	HTTPPort    = "8080"
	GRPCPort    = "8090"
	HTTPAddress = Host + ":" + HTTPPort
	GRPCAddress = Host + ":" + GRPCPort
)

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

func GetConfig() *config.Config {
	return &config.Config{
		Cookies:            nil,
		URL:                "http://" + HTTPAddress,
		WebSocketURL:       "ws://" + HTTPAddress,
		HTTPHeaders:        nil,
		TLSVerify:          false,
		Proxy:              "",
		MaxIdleConns:       2,
		MaxRedirects:       50,
		IdleConnTimeout:    2,
		FollowCookies:      false,
		BlockStatusCode:    403,
		PassStatusCode:     203,
		BlockRegex:         "",
		PassRegex:          "",
		NonBlockedAsPassed: false,
		Workers:            runtime.NumCPU(),
		RandomDelay:        400,
		SendDelay:          200,
		ReportPath:         path.Join(os.TempDir(), "reports"),
		TestCase:           "",
		TestCasesPath:      "",
		TestSet:            "",
		WAFName:            "test-waf",
		IgnoreUnresolved:   false,
		BlockConnReset:     false,
		SkipWAFBlockCheck:  false,
		AddHeader:          "",
	}
}

func GenerateTestCases() (testCases []test.Case, testCasesMap *TestCasesMap) {
	var encoders []string
	testCasesMap = new(TestCasesMap)
	testCasesMap.m = make(map[string]struct{})

	for encoderName, _ := range encoder.Encoders {
		if encoderName == encoder.DefaultGRPCEncoder.GetName() {
			continue
		}
		encoders = append(encoders, encoderName)
	}

	placeholders := []string{"Header", "RequestBody", "SOAPBody", "JSONRequest", "URLParam", "URLPath"}
	testSets := []string{"test-set1", "test-set2", "test-set3"}
	payloads := []string{"bypassed", "blocked", "unresolved"}

	for _, ts := range testSets {
		for _, ph := range placeholders {
			for _, enc := range encoders {
				name := fmt.Sprintf("%s-%s", ph, enc)
				testCases = append(testCases, test.Case{
					Payloads:       payloads,
					Encoders:       []string{enc},
					Placeholders:   []string{ph},
					Set:            ts,
					Name:           name,
					IsTruePositive: true,
				})
				for _, p := range payloads {
					testCasesMap.m[fmt.Sprintf("%s-%s-%s-%s-%s", ts, name, p, ph, enc)] = struct{}{}
				}
			}
		}
	}

	return
}
