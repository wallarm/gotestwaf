package db

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/wallarm/gotestwaf/internal/config"
	"github.com/wallarm/gotestwaf/internal/payload/placeholder"
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

		var t yamlConfig
		err = yaml.Unmarshal(yamlFile, &t)
		if err != nil {
			return nil, err
		}

		var placeholders []*Placeholder
		for _, ph := range t.Placeholders {
			switch typedPh := ph.(type) {
			case string:
				placeholders = append(placeholders, &Placeholder{Name: typedPh})

			case map[any]any:
				placeholderName := mapToString(typedPh)
				placeholderConfig, confErr := placeholder.GetPlaceholderConfig(placeholderName, typedPh[placeholderName])
				if confErr != nil {
					return nil, errors.Wrap(confErr, "couldn't parse config")
				}

				placeholders = append(placeholders, &Placeholder{
					Name:   placeholderName,
					Config: placeholderConfig,
				})

			default:
				return nil, errors.Errorf("couldn't parse config: unknown placeholder type, expected array of string or map[string]any, got %T", ph)
			}
		}

		testCase := &Case{
			Payloads:       t.Payloads,
			Encoders:       t.Encoders,
			Placeholders:   placeholders,
			Type:           t.Type,
			Set:            testSetName,
			Name:           testCaseName,
			IsTruePositive: true, // test case is true positive
		}

		if strings.Contains(testSetName, "false") {
			testCase.IsTruePositive = false // test case is false positive
		}

		testCases = append(testCases, testCase)
	}

	if testCases == nil {
		return nil, errors.New("no tests were selected")
	}

	return testCases, nil
}
