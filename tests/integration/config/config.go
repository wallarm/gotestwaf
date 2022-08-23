package config

import (
	"fmt"
	"net"
	"os"
	"path"
	"runtime"
	"sort"
	"sync"

	"github.com/wallarm/gotestwaf/internal/config"
	"github.com/wallarm/gotestwaf/internal/db"
	"github.com/wallarm/gotestwaf/internal/payload/encoder"
	"github.com/wallarm/gotestwaf/internal/payload/placeholder"
)

var (
	HTTPPort int
	GRPCPort int
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

func (tcm *TestCasesMap) GetRemainingValues() []string {
	var res []string

	tcm.Lock()
	defer tcm.Unlock()

	for k, _ := range tcm.m {
		res = append(res, k)
	}

	sort.Strings(res)

	return res
}

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func PickUpTestPorts() error {
	httpPort, err := getFreePort()
	if err != nil {
		return err
	}

	grpcPort, err := getFreePort()
	if err != nil {
		return err
	}

	HTTPPort = httpPort
	GRPCPort = grpcPort

	return nil
}

func GetConfig() *config.Config {
	return &config.Config{
		URL:                fmt.Sprintf("http://localhost:%d", HTTPPort),
		GRPCPort:           uint16(GRPCPort),
		WebSocketURL:       fmt.Sprintf("ws://localhost:%d", HTTPPort),
		HTTPHeaders:        nil,
		TLSVerify:          false,
		Proxy:              "",
		MaxIdleConns:       2,
		MaxRedirects:       50,
		IdleConnTimeout:    2,
		FollowCookies:      false,
		RenewSession:       false,
		BlockStatusCode:    403,
		PassStatusCode:     []int{200, 404},
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

func GenerateTestCases() (testCases []*db.Case, testCasesMap *TestCasesMap) {
	grpcEncoder := placeholder.DefaultGRPC.GetName()
	requestBodyPlaceholder := placeholder.DefaultRequestBody.GetName()

	var encoders []string
	var placeholders []string
	testCasesMap = new(TestCasesMap)
	testCasesMap.m = make(map[string]struct{})

	for encoderName, _ := range encoder.Encoders {
		encoders = append(encoders, encoderName)
	}

	for placeholderName, _ := range placeholder.Placeholders {
		placeholders = append(placeholders, placeholderName)
	}

	testSets := []string{"test-set1", "test-set2", "test-set3"}
	payloads := []string{"bypassed", "blocked", "unresolved"}

	for _, ts := range testSets {
		for _, ph := range placeholders {
			for _, enc := range encoders {
				if enc == grpcEncoder && ph != requestBodyPlaceholder {
					continue
				}

				name := fmt.Sprintf("%s-%s", ph, enc)
				testCases = append(testCases, &db.Case{
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
