package test

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/wallarm/gotestwaf/internal/data/config"
	"gopkg.in/yaml.v2"
)

func Load(cfg *config.Config, logger *log.Logger) ([]Case, error) {
	var files []string
	var testCases []Case

	if cfg.TestCasesPath == "" {
		return nil, errors.New("empty test cases path")
	}

	if err := filepath.Walk(cfg.TestCasesPath, func(path string, info os.FileInfo, err error) error {
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

		var t Case
		err = yaml.Unmarshal(yamlFile, &t)
		if err != nil {
			return nil, err
		}

		t.Name = testCaseName
		t.Set = testSetName
		if strings.Contains(testSetName, "false") {
			t.IsTruePositive = false // test case is false positive
		} else {
			t.IsTruePositive = true // test case is true positive
		}

		if cfg.TestCase != "" && t.Name != cfg.TestCase {
				continue
		}
		testCases = append(testCases, t)
	}

	return testCases, nil
}
