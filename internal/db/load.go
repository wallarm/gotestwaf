package db

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/wallarm/gotestwaf/internal/config"
)

func LoadTestCases(cfg *config.Config) (testCases []*Case, err error) {
	var files []string

	if cfg.TestCasesPath == "" {
		return nil, errors.New("empty test cases path")
	}

	if err = filepath.Walk(cfg.TestCasesPath, func(path string, info os.FileInfo, err error) error {
		files = append(files, path)
		return nil
	}); err != nil {
		return nil, err
	}

	for _, testCaseFile := range files {
		fileExt := filepath.Ext(testCaseFile)
		if fileExt != ".yml" && fileExt != ".yaml" {
			continue
		}

		// Ignore subdirectories, process as .../<testSetName>/<testCaseName>/<case>.yml
		parts := strings.Split(testCaseFile, string(os.PathSeparator))
		parts = parts[len(parts)-3:]

		testSetName := parts[1]
		testCaseName := strings.TrimSuffix(parts[2], fileExt)

		if cfg.TestSet != "" && testSetName != cfg.TestSet {
			continue
		}

		if cfg.TestCase != "" && testCaseName != cfg.TestCase {
			continue
		}

		yamlFile, err := os.ReadFile(testCaseFile)
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

		testCases = append(testCases, &t)
	}

	if testCases == nil {
		return nil, errors.New("no tests were selected")
	}

	return testCases, nil
}
