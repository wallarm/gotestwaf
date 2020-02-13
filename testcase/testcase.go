package testcase

import (
	"fmt"
	"gotestwaf/config"
	"gotestwaf/report"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
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

func Run(url string, config config.Config) report.Report {
	var wg sync.WaitGroup
	encoder.InitEncoders()
	testcases := Load("./testcases/")

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
						// TODO: Configure the way how to check results
						results.Lock.Lock()
						if ret.StatusCode == 403 {
							results.Report[testsetName][testcaseName][true]++
						} else {
							results.Report[testsetName][testcaseName][false]++
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
