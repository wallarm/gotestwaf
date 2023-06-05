package report

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"

	"github.com/wallarm/gotestwaf/internal/payload/encoder"
	"github.com/wallarm/gotestwaf/internal/payload/placeholder"
)

func TestCustomValidators(t *testing.T) {
	validate := validator.New()
	for tag, validatorFunc := range customValidators {
		err := validate.RegisterValidation(tag, validatorFunc)
		if err != nil {
			t.Fatalf("couldn't build validator: %v", err)
		}
	}

	report := &HtmlReport{
		Overall: &Grade{},
	}
	report.NegativeTests.Bypassed = map[string]map[string]map[int]*TestDetails{
		"path": {
			"payload": map[int]*TestDetails{
				200: {
					Encoders:     nil,
					Placeholders: nil,
				},
			},
		},
	}

	setGtwVersion := func(value any) { report.GtwVersion = value.(string) }
	setFp := func(value any) { report.TestCasesFP = value.(string) }
	setMark := func(value any) { report.Overall.Mark = value.(string) }
	setCssSuffix := func(value any) { report.Overall.CSSClassSuffix = value.(string) }
	setIndicator := func(value any) { report.ApiSecChartData.Indicators = []string{value.(string)} }
	setArgs := func(value any) { report.Args = strings.Split(value.(string), "|") }
	setEncoders := func(value any) {
		report.NegativeTests.Bypassed["path"]["payload"][200].Encoders = map[string]any{value.(string): nil}
	}
	setPlaceholders := func(value any) {
		report.NegativeTests.Bypassed["path"]["payload"][200].Placeholders = map[string]any{value.(string): nil}
	}

	type testCaseType struct {
		tag    string
		field  string
		setter func(value any)
		value  string
		isBad  bool
	}

	testCases := []testCaseType{
		// gtw_version, bad
		{tag: "gtw_version", field: "GtwVersion", setter: setGtwVersion, value: "", isBad: true},
		{tag: "gtw_version", field: "GtwVersion", setter: setGtwVersion, value: "lskdjflks", isBad: true},
		{tag: "gtw_version", field: "GtwVersion", setter: setGtwVersion, value: "v0", isBad: true},
		{tag: "gtw_version", field: "GtwVersion", setter: setGtwVersion, value: "v1.2.", isBad: true},

		// gtw_version, good
		{tag: "gtw_version", field: "GtwVersion", setter: setGtwVersion, value: "unknown", isBad: false},
		{tag: "gtw_version", field: "GtwVersion", setter: setGtwVersion, value: "v0.4.3", isBad: false},
		{tag: "gtw_version", field: "GtwVersion", setter: setGtwVersion, value: "v0.4.2-3-gf58cd99", isBad: false},

		// fp, bad
		{tag: "fp", field: "TestCasesFP", setter: setFp, value: "", isBad: true},
		{tag: "fp", field: "TestCasesFP", setter: setFp, value: "flkjadfl", isBad: true},
		{tag: "fp", field: "TestCasesFP", setter: setFp, value: "abcdefghijklmnopqrstuvwxyzabcdef", isBad: true},
		{tag: "fp", field: "TestCasesFP", setter: setFp, value: "0123456789abcdef", isBad: true},
		{tag: "fp", field: "TestCasesFP", setter: setFp, value: " 0123456789abcdef0123456789abcdef", isBad: true},
		{tag: "fp", field: "TestCasesFP", setter: setFp, value: "0123456789abcdef0123456789abcdef ", isBad: true},

		// fp, god
		{tag: "fp", field: "TestCasesFP", setter: setFp, value: "0123456789abcdef0123456789abcdef", isBad: false},

		// mark, bad
		{tag: "mark", field: "Overall.Mark", setter: setMark, value: "", isBad: true},
		{tag: "mark", field: "Overall.Mark", setter: setMark, value: "lakjdsf", isBad: true},
		{tag: "mark", field: "Overall.Mark", setter: setMark, value: "NA", isBad: true},
		{tag: "mark", field: "Overall.Mark", setter: setMark, value: "n/a", isBad: true},
		{tag: "mark", field: "Overall.Mark", setter: setMark, value: "a", isBad: true},
		{tag: "mark", field: "Overall.Mark", setter: setMark, value: "f", isBad: true},
		{tag: "mark", field: "Overall.Mark", setter: setMark, value: "G", isBad: true},
		{tag: "mark", field: "Overall.Mark", setter: setMark, value: "C++", isBad: true},
		{tag: "mark", field: "Overall.Mark", setter: setMark, value: "D--", isBad: true},
		{tag: "mark", field: "Overall.Mark", setter: setMark, value: " A", isBad: true},
		{tag: "mark", field: "Overall.Mark", setter: setMark, value: "F ", isBad: true},

		// mark, good
		{tag: "mark", field: "Overall.Mark", setter: setMark, value: "N/A", isBad: false},
		{tag: "mark", field: "Overall.Mark", setter: setMark, value: "A", isBad: false},
		{tag: "mark", field: "Overall.Mark", setter: setMark, value: "C", isBad: false},
		{tag: "mark", field: "Overall.Mark", setter: setMark, value: "F", isBad: false},
		{tag: "mark", field: "Overall.Mark", setter: setMark, value: "A+", isBad: false},
		{tag: "mark", field: "Overall.Mark", setter: setMark, value: "C+", isBad: false},
		{tag: "mark", field: "Overall.Mark", setter: setMark, value: "F-", isBad: false},

		// mark, bad
		{tag: "css_suffix", field: "Overall.CSSClassSuffix", setter: setCssSuffix, value: "", isBad: true},
		{tag: "css_suffix", field: "Overall.CSSClassSuffix", setter: setCssSuffix, value: "lakjdsf", isBad: true},
		{tag: "css_suffix", field: "Overall.CSSClassSuffix", setter: setCssSuffix, value: "NA", isBad: true},
		{tag: "css_suffix", field: "Overall.CSSClassSuffix", setter: setCssSuffix, value: "a+", isBad: true},
		{tag: "css_suffix", field: "Overall.CSSClassSuffix", setter: setCssSuffix, value: "b-", isBad: true},
		{tag: "css_suffix", field: "Overall.CSSClassSuffix", setter: setCssSuffix, value: "g", isBad: true},
		{tag: "css_suffix", field: "Overall.CSSClassSuffix", setter: setCssSuffix, value: "A", isBad: true},
		{tag: "css_suffix", field: "Overall.CSSClassSuffix", setter: setCssSuffix, value: "F", isBad: true},
		{tag: "css_suffix", field: "Overall.CSSClassSuffix", setter: setCssSuffix, value: " a", isBad: true},
		{tag: "css_suffix", field: "Overall.CSSClassSuffix", setter: setCssSuffix, value: "f ", isBad: true},

		// mark, good
		{tag: "css_suffix", field: "Overall.CSSClassSuffix", setter: setCssSuffix, value: "na", isBad: false},
		{tag: "css_suffix", field: "Overall.CSSClassSuffix", setter: setCssSuffix, value: "a", isBad: false},
		{tag: "css_suffix", field: "Overall.CSSClassSuffix", setter: setCssSuffix, value: "c", isBad: false},
		{tag: "css_suffix", field: "Overall.CSSClassSuffix", setter: setCssSuffix, value: "f", isBad: false},

		// indicator, bad
		{tag: "indicator", field: "ApiSecChartData.Indicators", setter: setIndicator, value: "", isBad: true},
		{tag: "indicator", field: "ApiSecChartData.Indicators", setter: setIndicator, value: "laksdjflk", isBad: true},
		{tag: "indicator", field: "ApiSecChartData.Indicators", setter: setIndicator, value: "laksdjflk", isBad: true},
		{tag: "indicator", field: "ApiSecChartData.Indicators", setter: setIndicator, value: "some indicator (1000.0%)", isBad: true},
		{tag: "indicator", field: "ApiSecChartData.Indicators", setter: setIndicator, value: "some indicator indicator indicator (10.0%)", isBad: true},

		// indicator, good
		{tag: "indicator", field: "ApiSecChartData.Indicators", setter: setIndicator, value: "-", isBad: false},
		{tag: "indicator", field: "ApiSecChartData.Indicators", setter: setIndicator, value: "some indicator (unavailable)", isBad: false},
		{tag: "indicator", field: "ApiSecChartData.Indicators", setter: setIndicator, value: "some indicator (0.0%)", isBad: false},
		{tag: "indicator", field: "ApiSecChartData.Indicators", setter: setIndicator, value: "some indicator (100.0%)", isBad: false},

		// args, bad
		{tag: "args", field: "Args", setter: setArgs, value: "", isBad: true},
		{tag: "args", field: "Args", setter: setArgs, value: "lkajdf", isBad: true},
		{tag: "args", field: "Args", setter: setArgs, value: "-a", isBad: true},
		{tag: "args", field: "Args", setter: setArgs, value: "-lkajdf", isBad: true},
		{tag: "args", field: "Args", setter: setArgs, value: "--lkajdf", isBad: true},
		{tag: "args", field: "Args", setter: setArgs, value: "--quiet|--url=url|--workers=10|--blockStatusCodes", isBad: true},
		{tag: "args", field: "Args", setter: setArgs, value: "--quiet|--url=url|--workers|--blockStatusCodes=403", isBad: true},
		{tag: "args", field: "Args", setter: setArgs, value: "--quiet|--url|--workers=10|--blockStatusCodes=403", isBad: true},
		{tag: "args", field: "Args", setter: setArgs, value: "--quiet=sdf|--url=url|--workers=10|--blockStatusCodes=403", isBad: true},
		{tag: "args", field: "Args", setter: setArgs, value: "--quiet sdf", isBad: true},
		{tag: "args", field: "Args", setter: setArgs, value: "--url url", isBad: true},
		{tag: "args", field: "Args", setter: setArgs, value: "--workers 10", isBad: true},
		{tag: "args", field: "Args", setter: setArgs, value: "--blockStatusCodes 403", isBad: true},

		// args, good
		{tag: "args", field: "Args", setter: setArgs, value: "--quiet", isBad: false},
		{tag: "args", field: "Args", setter: setArgs, value: "--url=url", isBad: false},
		{tag: "args", field: "Args", setter: setArgs, value: "--workers=10", isBad: false},
		{tag: "args", field: "Args", setter: setArgs, value: "--blockStatusCodes=403,401", isBad: false},
		{tag: "args", field: "Args", setter: setArgs, value: "--quiet|--url=url|--workers=10|--blockStatusCodes=403,401", isBad: false},

		// encoders, bad
		{tag: "encoders", field: "NegativeTests.Bypassed[path][payload][200].Encoders", setter: setEncoders, value: "", isBad: true},
		{tag: "encoders", field: "NegativeTests.Bypassed[path][payload][200].Encoders", setter: setEncoders, value: "unknown", isBad: true},

		// placeholders, bad
		{tag: "placeholders", field: "NegativeTests.Bypassed[path][payload][200].Placeholders", setter: setPlaceholders, value: "", isBad: true},
		{tag: "placeholders", field: "NegativeTests.Bypassed[path][payload][200].Placeholders", setter: setPlaceholders, value: "unknown", isBad: true},
	}

	// encoders, good
	for enc, _ := range encoder.Encoders {
		testCases = append(testCases, testCaseType{tag: "encoders", field: "NegativeTests.Bypassed[path][payload][200].Encoders", setter: setEncoders, value: enc, isBad: false})
	}

	// placeholders, good
	for ph, _ := range placeholder.Placeholders {
		testCases = append(testCases, testCaseType{tag: "placeholders", field: "NegativeTests.Bypassed[path][payload][200].Placeholders", setter: setPlaceholders, value: ph, isBad: false})
	}

	var err error
	var errMsg string
	var tag string
	var value string
	var values []string
	var contains bool

	for _, testCase := range testCases {
		tag = testCase.tag
		value = testCase.value

		testCase.setter(testCase.value)

		err = validate.StructPartial(report, testCase.field)

		if err != nil {
			err = &ValidationError{err.(validator.ValidationErrors)}
			errMsg = err.Error()

			if testCase.isBad {
				values = []string{value}
				if strings.Contains(value, "|") {
					values = strings.Split(value, "|")
				}

				contains = false
				for _, val := range values {
					if strings.Contains(errMsg, val) {
						contains = true
						break
					}
				}

				if !contains {
					t.Errorf("%s: error msg doesn't contain bad value, bad value: '%s', err msg: '%s'", tag, value, errMsg)
				}
			} else {
				t.Errorf("%s: false positive, value: '%s', err msg: '%s'", tag, value, errMsg)
			}
		}

		if err == nil && testCase.isBad {
			t.Errorf("%s: bad validation, bad value: '%s', error: %v", tag, value, err)
		}
	}
}
