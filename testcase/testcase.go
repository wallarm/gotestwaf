package testcase

import (
	"fmt"
	"gotestwaf/config"
	"gotestwaf/report"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"gotestwaf/payload"
	"gotestwaf/payload/encoder"

	"gopkg.in/yaml.v2"
)

type Testcase struct {
	Payloads     []string `yaml:"payload"`
	Encoders     []string `yaml:"encoder"`
	Placeholders []string `yaml:"placeholder"`
	Testset      string
	Name         string
}

func Load(testcaseFolder string) []Testcase {
	var files []string
	var testcases []Testcase

	if err := filepath.Walk(testcaseFolder, func(path string, info os.FileInfo, err error) error {
		files = append(files, path)
		return nil
	}); err != nil {
		panic(err)
	}

	fmt.Println("Loading testcases: ")
	for _, testcaseFile := range files {

		if filepath.Ext(testcaseFile) != ".yml" {
			continue
		}

		parts := strings.Split(testcaseFile, "/")
		testsetName := parts[1]
		testcaseName := strings.TrimSuffix(parts[2], path.Ext(parts[2]))

		fmt.Printf("%v\t%v\n", testsetName, testcaseName)

		if yamlFile, err := ioutil.ReadFile(testcaseFile); err != nil {
			log.Printf("yamlFile.Get err   #%v ", err)
		} else {
			testcase := Testcase{}
			if err = yaml.Unmarshal(yamlFile, &testcase); err != nil {
				log.Printf("Unmarshal: %v", err)
			} else {
				testcase.Name = testcaseName
				testcase.Testset = testsetName
				testcases = append(testcases, testcase)
			}
		}
	}

	return testcases
}

func CheckBlocking(resp *http.Response, config config.Config) (bool, int) {
	if config.BlockRegExp != "" {
		respData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		m, _ := regexp.MatchString(config.BlockRegExp, string(respData))
		return m, resp.StatusCode
	}
	return (resp.StatusCode == config.BlockStatusCode), resp.StatusCode
}

func PreCheck(url string, config config.Config) (bool, int) {
	encoder.InitEncoders()
	ret := payload.Send(config, url, "UrlParam", "Url", "<script>alert('union select password from users')</script>")
	return CheckBlocking(ret, config)
}

func Run(url string, config config.Config) report.Report {
	var wg sync.WaitGroup
	encoder.InitEncoders()
	testcases := Load(config.TestcasesFolder)

	results := report.CreateReport()

	for _, testcase := range testcases {
		if results.Report[testcase.Testset] == nil {
			results.Report[testcase.Testset] = map[string]map[bool]int{}
		}
		if results.Report[testcase.Testset][testcase.Name] == nil {
			results.Report[testcase.Testset][testcase.Name] = map[bool]int{}
		}
		results.Report[testcase.Testset][testcase.Name][true] = 0
		results.Report[testcase.Testset][testcase.Name][false] = 0
		for _, payloadData := range testcase.Payloads {
			for _, encoderName := range testcase.Encoders {
				for _, placeholder := range testcase.Placeholders {
					wg.Add(1)
					go func(testsetName string, testcaseName string, payloadData string, encoderName string, placeholder string, wg *sync.WaitGroup) {
						defer wg.Done()
						ret := payload.Send(config, url, placeholder, encoderName, payloadData)
						results.Lock.Lock()
						blocked, _ := CheckBlocking(ret, config)
						if blocked {
							results.Report[testsetName][testcaseName][true]++
						} else {
							results.Report[testsetName][testcaseName][false]++
							test := report.Test{Testset: testsetName, Testcase: testcaseName, Payload: payloadData, Encoder: encoderName, Placeholder: placeholder}
							results.FailedTests = append(results.FailedTests, test)
						}
						results.Lock.Unlock()
						fmt.Printf(".")
					}(testcase.Testset, testcase.Name, payloadData, encoderName, placeholder, &wg)
				}
			}
		}
	}
	wg.Wait()
	fmt.Printf("\n")
	return results
}
