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
	"github.com/wallarm/gotestwaf/internal/payload/placeholder"
)

var placeholdersEncodersMap = map[string][]string{
	"gRPC":              {"Base64", "Base64Flat", "JSUnicode", "Plain", "URL", "XMLEntity"},
	"Header":            {"Base64", "Base64Flat", "JSUnicode", "Plain", "URL", "XMLEntity"},
	"HTMLForm":          {"Base64", "Base64Flat", "Plain", "URL"},
	"HTMLMultipartForm": {"Base64", "Base64Flat", "Plain", "URL"},
	"JSONBody":          {"Base64", "Base64Flat", "JSUnicode", "Plain", "URL", "XMLEntity"},
	"JSONRequest":       {"Plain"},
	"RequestBody":       {"Base64", "Base64Flat", "JSUnicode", "Plain", "URL", "XMLEntity"},
	"SOAPBody":          {"Base64", "Base64Flat", "JSUnicode", "Plain", "URL", "XMLEntity"},
	"URLParam":          {"Base64", "Base64Flat", "Plain", "URL"},
	"URLPath":           {"Base64", "Base64Flat", "Plain", "URL"},
	"UserAgent":         {"Base64", "Base64Flat", "JSUnicode", "Plain", "URL", "XMLEntity"},
	"XMLBody":           {"Base64", "Base64Flat", "XMLEntity"},
}

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

func PickUpTestPorts() (httpPort, grpcPort int, err error) {
	httpPort, err = getFreePort()
	if err != nil {
		return 0, 0, err
	}

	grpcPort, err = getFreePort()
	if err != nil {
		return 0, 0, err
	}

	return
}

func getConfig(httpPort int, grpcPort int) *config.Config {
	return &config.Config{
		// Target settings
		URL:         fmt.Sprintf("http://localhost:%d", httpPort),
		GRPCPort:    uint16(grpcPort),
		GraphQLURL:  fmt.Sprintf("http://localhost:%d/graphql", httpPort),
		OpenAPIFile: "",

		// Test cases settings
		TestCase:      "",
		TestCasesPath: "",
		TestSet:       "",

		// HTTP client settings
		HTTPClient:     "", // will be changed to 'gohttp' or 'chrome'
		TLSVerify:      false,
		Proxy:          "",
		AddHeader:      "",
		AddDebugHeader: true,

		// GoHTTP client only settings
		MaxIdleConns:    2,
		MaxRedirects:    50,
		IdleConnTimeout: 2,
		FollowCookies:   false,
		RenewSession:    false,

		// Performance settings
		Workers:     runtime.NumCPU(),
		RandomDelay: 400,
		SendDelay:   200,

		// Analysis settings
		SkipWAFBlockCheck:     false,
		SkipWAFIdentification: true,
		BlockStatusCodes:      []int{403},
		PassStatusCodes:       []int{200, 404},
		BlockRegex:            "",
		PassRegex:             "",
		NonBlockedAsPassed:    false,
		IgnoreUnresolved:      false,
		BlockConnReset:        false,

		// Report settings
		WAFName:         "test-waf",
		IncludePayloads: false,
		ReportPath:      path.Join(os.TempDir(), "reports"),
		ReportName:      "test",
		ReportFormat:    []string{""},
		NoEmailReport:   true,
		Email:           "",

		// config.yaml
		HTTPHeaders: map[string]string{},

		// Other settings
		LogLevel: "debug",

		CheckBlockFunc: nil,

		Args: nil,
	}
}

func GetConfigWithGoHTTPClient(httpPort int, grpcPort int) *config.Config {
	cfg := getConfig(httpPort, grpcPort)

	cfg.HTTPClient = "gohttp"
	cfg.HTTPHeaders["client"] = "gohttp"

	return cfg
}

func GetConfigWithChromeClient(httpPort int, grpcPort int) *config.Config {
	cfg := getConfig(httpPort, grpcPort)

	cfg.HTTPClient = "chrome"
	cfg.HTTPHeaders["client"] = "chrome"

	return cfg
}

func GenerateTestCases() (testCases []*db.Case, testCasesMap *TestCasesMap) {
	testCasesMap = new(TestCasesMap)
	testCasesMap.m = make(map[string]string)

	testSets := []string{"test-set1", "test-set2", "test-set3"}
	payloads := []string{"bypassed", "blocked", "unresolved"}

	var debugHeader string

	hash := sha256.New()

	f := func(testSet string, payloads []string, encoder string, placeholderName string, placeholderConf placeholder.PlaceholderConfig) {
		name := fmt.Sprintf("%s-%s", placeholderName, encoder)
		testCases = append(testCases, &db.Case{
			Payloads: payloads,
			Encoders: []string{encoder},
			Placeholders: []*db.Placeholder{{
				Name:   placeholderName,
				Config: placeholderConf,
			}},
			Set:            testSet,
			Name:           name,
			IsTruePositive: true,
		})

		for _, payload := range payloads {
			hash.Reset()

			hash.Write([]byte(testSet))
			hash.Write([]byte(name))
			hash.Write([]byte(placeholderName))
			hash.Write([]byte(encoder))
			hash.Write([]byte(payload))

			debugHeader = hex.EncodeToString(hash.Sum(nil))

			testCasesMap.m[debugHeader] = fmt.Sprintf(
				"set=%s,name=%s,placeholder=%s,encoder=%s",
				testSet, name, placeholderName, encoder,
			)
		}
	}

	for _, testSet := range testSets {
		for placeholder, encoders := range placeholdersEncodersMap {
			for _, encoder := range encoders {
				f(testSet, payloads, encoder, placeholder, nil)
			}
		}
	}

	for testSet, settings := range RawRequestConfigs {
		for _, encoder := range settings.Encoders {
			f(testSet, payloads, encoder, placeholder.DefaultRawRequest.GetName(), settings.Config)
		}
	}

	for testSet, settings := range GraphQLConfigs {
		for _, encoder := range settings.Encoders {
			f(testSet, payloads, encoder, placeholder.DefaultGraphQL.GetName(), settings.Config)
		}
	}

	return
}
