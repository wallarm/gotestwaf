package testcase

import (
	"fmt"
	"gotestwaf/config"
	"gotestwaf/payload"
	"gotestwaf/payload/encoder"
	"gotestwaf/report"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

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

func Run(url string, config *config.Config) *report.Report {
	encoder.InitEncoders()
	testcases := Load("./testcases/")
	results := report.CreateReport()

	for _, testcase := range testcases {
		for _, payloadData := range testcase.Payloads {
			for _, encoderName := range testcase.Encoders {
				for _, placeholder := range testcase.Placeholders {
					rk := report.ReportKey{
						Testset: testcase.Testset,
						Name:    testcase.Name,
					}
					results.Report[rk] = map[bool]int{true: 0, false: 0}
					results.Wg.Add(1)
					go TestExecutor(&url, config, rk, payloadData, encoderName, placeholder, results)
				}
			}
		}
	}
	results.Wg.Wait()
	return results
}

func TestExecutor(url *string, cfg *config.Config, reportKey report.ReportKey, payloadData string, encoderName string, placeholder string, res *report.Report) {
	defer res.Wg.Done()
	ret := payload.Send(cfg, *url, placeholder, encoderName, payloadData)
	// TODO: Configure the way how to check results
	res.Lock.Lock()
	if ret.StatusCode == 403 {
		res.Report[reportKey][true]++
	} else {
		res.Report[reportKey][false]++
	}
	res.Lock.Unlock()
	fmt.Printf(".")
}
