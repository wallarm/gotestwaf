package test

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"github.com/wallarm/gotestwaf/internal/data/config"
)

const testCaseExt = ".yml"

func Load(cfg *config.Config, logger *log.Logger) ([]Case, error) {
	var files []string
	var testCases []Case
	var loadBindData = false

	if cfg.TestCasesPath == "" {
		return nil, errors.New("empty test cases path")
	}
	
	if _, err := os.Stat(cfg.TestCasesPath); err != nil {
		for path, _ := range _bindata {
			if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".png") {
				continue
			}
			files = append(files, path)
		}
		loadBindData = true
	} else {
		if err := filepath.Walk(cfg.TestCasesPath, func(path string, info os.FileInfo, err error) error {
			files = append(files, path)
			return nil
		}); err != nil {
			return nil, err
		}
	}
	// bugfix: fix array out of bounds error caused by separator on windows platform
	pathSeparator := string(os.PathSeparator)
	if loadBindData {
		pathSeparator = "/"
	}

	for _, testCaseFile := range files {
		if filepath.Ext(testCaseFile) != testCaseExt {
			continue
		}

		// Ignore subdirectories, process as .../<testSetName>/<testCaseName>/<case>.yml
		parts := strings.Split(testCaseFile, pathSeparator)
		parts = parts[len(parts)-3:]

		testSetName := parts[1]
		testCaseName := strings.TrimSuffix(parts[2], testCaseExt)

		if cfg.TestSet != "" && testSetName != cfg.TestSet {
			continue
		}

		if cfg.TestCase != "" && testCaseName != cfg.TestCase {
			continue
		}
		
		var yamlFile []byte
		var err error
		if loadBindData {
			yamlFile, err = Asset(testCaseFile)
		} else {
			yamlFile, err = ioutil.ReadFile(testCaseFile)
		}
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

		testCases = append(testCases, t)
	}

	if testCases == nil {
		return nil, errors.New("no tests were selected")
	}

	return testCases, nil
}
