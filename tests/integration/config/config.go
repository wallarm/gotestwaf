package config

import (
	"crypto/sha256"
	"encoding/hex"
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
	m map[string]string
}

func (tcm *TestCasesMap) CheckTestCaseAvailability(caseHash string) (string, bool) {
	tcm.Lock()
	defer tcm.Unlock()

	if value, ok := tcm.m[caseHash]; ok {
		delete(tcm.m, caseHash)
		return value, true
	}

	return "", false
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
		GraphQlURL:         fmt.Sprintf("http://localhost:%d/graphql", HTTPPort),
		HTTPHeaders:        nil,
		TLSVerify:          false,
		Proxy:              "",
		MaxIdleConns:       2,
		MaxRedirects:       50,
		IdleConnTimeout:    2,
		FollowCookies:      false,
		RenewSession:       false,
		BlockStatusCodes:   []int{403},
		PassStatusCodes:    []int{200, 404},
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
		AddDebugHeader:     true,
	}
}

func GenerateTestCases() (testCases []*db.Case, testCasesMap *TestCasesMap) {
	var encoders []string
	var placeholders []string
	testCasesMap = new(TestCasesMap)
	testCasesMap.m = make(map[string]string)

	for encoderName, _ := range encoder.Encoders {
		encoders = append(encoders, encoderName)
	}

	for placeholderName, _ := range placeholder.Placeholders {
		placeholders = append(placeholders, placeholderName)
	}

	testSets := []string{"test-set1", "test-set2", "test-set3"}
	payloads := []string{"bypassed", "blocked", "unresolved"}

	var debugHeader string

	hash := sha256.New()

	for _, testSet := range testSets {
		for _, placeholder := range placeholders {
			for _, encoder := range encoders {
				name := fmt.Sprintf("%s-%s", placeholder, encoder)
				testCases = append(testCases, &db.Case{
					Payloads:       payloads,
					Encoders:       []string{encoder},
					Placeholders:   []string{placeholder},
					Set:            testSet,
					Name:           name,
					IsTruePositive: true,
				})

				for _, payload := range payloads {
					hash.Reset()

					hash.Write([]byte(testSet))
					hash.Write([]byte(name))
					hash.Write([]byte(placeholder))
					hash.Write([]byte(encoder))
					hash.Write([]byte(payload))

					debugHeader = hex.EncodeToString(hash.Sum(nil))

					testCasesMap.m[debugHeader] = fmt.Sprintf(
						"set=%s,name=%s,placeholder=%s,encoder=%s",
						testSet, name, placeholder, encoder,
					)
				}
			}
		}
	}

	return
}
