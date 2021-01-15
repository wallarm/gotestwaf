package scanner

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/wallarm/gotestwaf/config"
	"github.com/wallarm/gotestwaf/payload"
	"github.com/wallarm/gotestwaf/payload/encoder"
	"github.com/wallarm/gotestwaf/report"
)

type Scanner struct {
	logger *log.Logger
	cfg    *config.Config
}

func New(cfg *config.Config, logger *log.Logger) *Scanner {
	return &Scanner{
		cfg:    cfg,
		logger: logger,
	}
}

func (s *Scanner) CheckBlocking(resp *http.Response) (bool, int, error) {
	if s.cfg.BlockRegExp != "" {
		respData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, 0, err
		}
		m, _ := regexp.MatchString(s.cfg.BlockRegExp, string(respData))
		return m, resp.StatusCode, nil
	}
	return resp.StatusCode == s.cfg.BlockStatusCode, resp.StatusCode, nil
}

func (s *Scanner) CheckPass(resp *http.Response) (bool, int, error) {
	if s.cfg.PassRegExp != "" {
		respData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, 0, err
		}
		m, _ := regexp.MatchString(s.cfg.PassRegExp, string(respData))
		return m, resp.StatusCode, nil
	}
	return resp.StatusCode == s.cfg.PassStatusCode, resp.StatusCode, nil
}

func (s *Scanner) PreCheck(url string) (bool, int, error) {
	encoder.InitEncoders()
	ret := payload.Send(s.cfg,
		url,
		"URLParam",
		"URL",
		"<script>alert('union select password from users')</script>",
	)
	return s.CheckBlocking(ret)
}

func (s *Scanner) Run(url string) (*report.Report, error) {
	s.logger.Println("Test cases loading started")
	testcases, err := Load(s.cfg.TestCasesFolder, s.logger)
	if err != nil {
		return nil, err
	}
	s.logger.Println("Test cases loading finished")

	results := report.New()
	encoder.InitEncoders()
	var wg sync.WaitGroup
	s.logger.Println("Scanning started")

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
						time.Sleep(time.Duration(s.cfg.SendingDelay+rand.Intn(s.cfg.RandomDelay)) * time.Millisecond)
						ret := payload.Send(s.cfg, url, placeholder, encoderName, payloadData)
						results.Mu.Lock()
						blocked, _, err := s.CheckBlocking(ret)
						if err != nil {
							s.logger.Println("failed to check blocking:", err)
						}
						passed, _, err := s.CheckPass(ret)
						if err != nil {
							s.logger.Println("failed to check pass:", err)
						}
						if (blocked && passed) || (!blocked && !passed) {
							results.Report[testSetName][testCaseName][s.cfg.NonBlockedAsPassed]++
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
	s.logger.Println("Scanning finished")

	return results, nil
}
