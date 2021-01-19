package scanner

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
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

func Load(testCasesPath string, logger *log.Logger) ([]TestCase, error) {
	var files []string
	var testCases []TestCase

	if testCasesPath == "" {
		return nil, errors.New("empty test cases path")
	}

	if err := filepath.Walk(testCasesPath, func(path string, info os.FileInfo, err error) error {
		files = append(files, path)
		return nil
	}); err != nil {
		return nil, err
	}

	logger.Println("Loading test cases: ")
	for _, testCaseFile := range files {
		if filepath.Ext(testCaseFile) != ".yml" {
			continue
		}

		parts := strings.Split(testCaseFile, "/")
		testSetName := parts[1]
		testCaseName := strings.TrimSuffix(parts[2], path.Ext(parts[2]))

		logger.Printf("%v:%v", testSetName, testCaseName)

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
