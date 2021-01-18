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

	"github.com/pkg/errors"
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
	testCases, err := Load(s.cfg.TestCasesPath, s.logger)
	if err != nil {
		return nil, errors.Wrap(err, "loading test cases")
	}
	s.logger.Println("Test cases loading finished")

	results := report.New()
	encoder.InitEncoders()
	var wg sync.WaitGroup
	s.logger.Println("Scanning started")

	for _, tc := range testCases {
		if results.Report[tc.TestSet] == nil {
			results.Report[tc.TestSet] = map[string]map[bool]int{}
		}
		if results.Report[tc.TestSet][tc.Name] == nil {
			results.Report[tc.TestSet][tc.Name] = map[bool]int{}
		}
		results.Report[tc.TestSet][tc.Name][true] = 0
		results.Report[tc.TestSet][tc.Name][false] = 0
		for _, payloadData := range tc.Payloads {
			for _, encoderName := range tc.Encoders {
				for _, placeholder := range tc.Placeholders {
					wg.Add(1)
					go func(testSetName string, testCaseName string, payloadData string, encoderName string, placeholder string, wg *sync.WaitGroup) {
						defer wg.Done()
						time.Sleep(time.Duration(s.cfg.SendDelay+rand.Intn(s.cfg.RandomDelay)) * time.Millisecond)
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
							if (blocked && tc.Type) ||
								// /*true negatives for malicious payloads (Type is true) and false positives checks (Type is false)
								(!blocked && !tc.Type) {
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
					}(tc.TestSet, tc.Name, payloadData, encoderName, placeholder, &wg)
				}
			}
		}
	}
	wg.Wait()
	fmt.Printf("\n")
	s.logger.Println("Scanning finished")

	return results, nil
}
