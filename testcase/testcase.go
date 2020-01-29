package testcase

import (
	"fmt"
	"gotestwaf/config"
	"gotestwaf/report"
	"io/ioutil"
	"log"
	"os"
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

	err := filepath.Walk(testcaseFolder, func(path string, info os.FileInfo, err error) error {
		files = append(files, path)
		return nil
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Loading testcases: ")
	for _, testcaseFile := range files {

		if filepath.Ext(testcaseFile) != ".yml" {
			continue
		}

		parts := strings.Split(testcaseFile, "/")
		testsetName := parts[1]
		testcaseName := parts[2]

		fmt.Printf("%v\t%v\n", testsetName, testcaseName)

		yamlFile, err := ioutil.ReadFile(testcaseFile)
		if err != nil {
			log.Printf("yamlFile.Get err   #%v ", err)
			continue
		}

		testcase := Testcase{Testset: testsetName, Name: testcaseName}

		err = yaml.Unmarshal(yamlFile, &testcase)
		if err != nil {
			log.Fatalf("Unmarshal: %v", err)
			continue
		}

		testcases = append(testcases, testcase)

	}

	return testcases
}

func Run(url string, config config.Config) map[string]map[string]map[bool]int {
	var wg sync.WaitGroup
	encoder.InitEncoders()
	testcases := Load("./testcases/")

	results := report.Report{}
	// m["var1"] = map[string]string{}
	// m["var1"]["var2"] = "something"
	// fmt.Println(m["var1"]["var2"])

	for _, testcase := range testcases {
		fmt.Printf("%+v\n", testcase)
		results[testcase.Testset] = map[string]map[bool]int{}
		results[testcase.Testset][testcase.Name] = map[bool]int{}
		results[testcase.Testset][testcase.Name][true] = 0
		results[testcase.Testset][testcase.Name][false] = 0
		for _, payloadData := range testcase.Payloads {
			for _, encoderName := range testcase.Encoders {
				for _, placeholder := range testcase.Placeholders {
					wg.Add(1)
					go func(payloadData string, encoderName string, placeholder string, wg *sync.WaitGroup) {
						var result string
						defer wg.Done()
						ret := payload.Send(config, url, placeholder, encoderName, payloadData)
						// TODO: Configure the way how to check results
						if ret.StatusCode == 403 {
							result = "OK\n"
							results[testcase.Testset][testcase.Name][true]++
						} else {
							result = "FAIL\n"
							results[testcase.Testset][testcase.Name][false]++
						}
						fmt.Printf("Test %v %v : %v / %v / %v : %s", testcase.Testset, testcase.Name, payloadData, encoderName, placeholder, result)
					}(payloadData, encoderName, placeholder, &wg)
				}
			}
		}
	}
	wg.Wait()
	return results
}
