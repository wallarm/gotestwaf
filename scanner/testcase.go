package testcase

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/wallarm/gotestwaf/config"
	"github.com/wallarm/gotestwaf/payload"
	"github.com/wallarm/gotestwaf/payload/encoder"
	"github.com/wallarm/gotestwaf/report"
	"gopkg.in/yaml.v2"
)

type TestCase struct {
	Payloads     []string `yaml:"payload"`
	Encoders     []string `yaml:"encoder"`
	Placeholders []string `yaml:"placeholder"`
	TestSet      string
	Name         string
	Type         bool
}

func Load(testCaseFolder string) ([]TestCase, error) {
	var files []string
	var testCases []TestCase

	if err := filepath.Walk(testCaseFolder, func(path string, info os.FileInfo, err error) error {
		files = append(files, path)
		return nil
	}); err != nil {
		return nil, err
	}

	fmt.Println("Loading test cases: ")
	for _, testCaseFile := range files {
		if filepath.Ext(testCaseFile) != ".yml" {
			continue
		}

		parts := strings.Split(testCaseFile, "/")
		testSetName := parts[1]
		testCaseName := strings.TrimSuffix(parts[2], path.Ext(parts[2]))

		fmt.Printf("%v\t%v\n", testSetName, testCaseName)

		yamlFile, err := ioutil.ReadFile(testCaseFile)
		if err != nil {
			return nil, err
		}
		var testCase TestCase
		if err = yaml.Unmarshal(yamlFile, &testCase); err != nil {
			return nil, err
		}

		testCase.Name = testCaseName
		testCase.TestSet = testSetName
		if strings.Contains(testSetName, "false") {
			testCase.Type = false // test case is false positive
		} else {
			testCase.Type = true // test case is true positive
		}
		testCases = append(testCases, testCase)
	}

	return testCases, nil
}

func CheckBlocking(resp *http.Response, cfg *config.Config) (bool, int) {
	if cfg.BlockRegExp != "" {
		respData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		m, _ := regexp.MatchString(cfg.BlockRegExp, string(respData))
		return m, resp.StatusCode
	}
	return resp.StatusCode == cfg.BlockStatusCode, resp.StatusCode
}

func CheckPass(resp *http.Response, cfg *config.Config) (bool, int) {
	if cfg.PassRegExp != "" {
		respData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		m, _ := regexp.MatchString(cfg.PassRegExp, string(respData))
		return m, resp.StatusCode
	}
	return resp.StatusCode == cfg.PassStatusCode, resp.StatusCode
}

func PreCheck(url string, cfg *config.Config) (bool, int) {
	encoder.InitEncoders()
	ret := payload.Send(cfg,
		url,
		"URLParam",
		"URL",
		"<script>alert('union select password from users')</script>",
	)
	return CheckBlocking(ret, cfg)
}

func Run(url string, cfg *config.Config) (*report.Report, error) {
	var wg sync.WaitGroup
	encoder.InitEncoders()
	testcases, err := Load(cfg.TestCasesFolder)
	if err != nil {
		return nil, err
	}

	results := report.CreateReport()

	for _, testCase := range testcases {
		if results.Report[testCase.TestSet] == nil {
			results.Report[testCase.TestSet] = map[string]map[bool]int{}
		}
		if results.Report[testCase.TestSet][testCase.Name] == nil {
			results.Report[testCase.TestSet][testCase.Name] = map[bool]int{}
		}
		results.Report[testCase.TestSet][testCase.Name][true] = 0
		results.Report[testCase.TestSet][testCase.Name][false] = 0
		for _, payloadData := range testCase.Payloads {
			for _, encoderName := range testCase.Encoders {
				for _, placeholder := range testCase.Placeholders {
					wg.Add(1)
					go func(testSetName string, testCaseName string, payloadData string, encoderName string, placeholder string, wg *sync.WaitGroup) {
						defer wg.Done()
						time.Sleep(time.Duration(cfg.SendingDelay+rand.Intn(cfg.RandomDelay)) * time.Millisecond)
						ret := payload.Send(cfg, url, placeholder, encoderName, payloadData)
						results.Mu.Lock()
						blocked, _ := CheckBlocking(ret, cfg)
						passed, _ := CheckPass(ret, cfg)
						if (blocked && passed) || (!blocked && !passed) {
							results.Report[testSetName][testCaseName][cfg.NonBlockedAsPassed]++
							test := report.Test{
								TestSet:     testSetName,
								TestCase:    testCaseName,
								Payload:     payloadData,
								Encoder:     encoderName,
								Placeholder: placeholder,
								StatusCode:  ret.StatusCode,
							}
							results.NaTests = append(results.NaTests, test)
						} else {
							// true positives
							if (blocked && testCase.Type) ||
								// /*true negatives for malicious payloads (Type is true) and false positives checks (Type is false)
								(!blocked && !testCase.Type) {
								results.Report[testSetName][testCaseName][true]++
								test := report.Test{
									TestSet:     testSetName,
									TestCase:    testCaseName,
									Payload:     payloadData,
									Encoder:     encoderName,
									Placeholder: placeholder,
									StatusCode:  ret.StatusCode,
								}
								results.PassedTests = append(results.PassedTests, test)
							} else {
								results.Report[testSetName][testCaseName][false]++
								test := report.Test{
									TestSet:     testSetName,
									TestCase:    testCaseName,
									Payload:     payloadData,
									Encoder:     encoderName,
									Placeholder: placeholder,
									StatusCode:  ret.StatusCode,
								}
								results.FailedTests = append(results.FailedTests, test)
							}
						}
						results.Mu.Unlock()
						fmt.Printf(".")
					}(testCase.TestSet, testCase.Name, payloadData, encoderName, placeholder, &wg)
				}
			}
		}
	}
	wg.Wait()
	fmt.Printf("\n")

	return &results, nil
}
