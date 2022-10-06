package report

import (
	"bytes"
	_ "embed"
	"html/template"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/db"
	"github.com/wallarm/gotestwaf/internal/version"
)

const naMark = "N/A"

//go:embed report_template.html
var htmlTemplate string

type grade struct {
	Percentage  float32
	Mark        string
	ClassSuffix string
}

type comparisonTableRow struct {
	Name         string
	ApiSec       grade
	AppSec       grade
	OverallScore grade
}

type testDetails struct {
	TestCase     string
	Encoders     map[string]any
	Placeholders map[string]any
}

type testSetSummary struct {
	TestCases []*db.SummaryTableRow

	Percentage float64
	Sent       int
	Blocked    int
	Bypassed   int
	Unresolved int
	Failed     int

	resolvedTestCasesNumber int
}

type htmlReport struct {
	IgnoreUnresolved bool

	WafName        string
	Url            string
	WafTestingDate string
	GtwVersion     string
	TestCasesFP    string
	OpenApiFile    string
	Args           string

	ApiChartScript *template.HTML
	AppChartScript *template.HTML

	Overall grade
	ApiSec  struct {
		TrueNegative grade
		TruePositive grade
		Grade        grade
	}
	AppSec struct {
		TrueNegative grade
		TruePositive grade
		Grade        grade
	}

	ComparisonTable []*comparisonTableRow

	TotalSent                int
	BlockedRequestsNumber    int
	BypassedRequestsNumber   int
	UnresolvedRequestsNumber int
	FailedRequestsNumber     int

	ScannedPaths db.ScannedPaths

	NegativeTests struct {
		SummaryTable map[string]*testSetSummary

		// map[paths]map[payload]map[statusCode]*testDetails
		Bypassed map[string]map[string]map[int]*testDetails
		// map[payload]map[statusCode]*testDetails
		Unresolved map[string]map[int]*testDetails
		Failed     []*db.FailedDetails

		Percentage               float64
		TotalSent                int
		BlockedRequestsNumber    int
		BypassedRequestsNumber   int
		UnresolvedRequestsNumber int
		FailedRequestsNumber     int
	}

	PositiveTests struct {
		SummaryTable map[string]*testSetSummary

		// map[payload]map[statusCode]*testDetails
		Blocked map[string]map[int]*testDetails
		// map[payload]map[statusCode]*testDetails
		Bypassed map[string]map[int]*testDetails
		// map[payload]map[statusCode]*testDetails
		Unresolved map[string]map[int]*testDetails
		Failed     []*db.FailedDetails

		Percentage               float64
		TotalSent                int
		BlockedRequestsNumber    int
		BypassedRequestsNumber   int
		UnresolvedRequestsNumber int
		FailedRequestsNumber     int
	}
}

func isApiTest(setName string) bool {
	return strings.Contains(setName, "api")
}

func computeGrade(value float32, all int) grade {
	g := grade{
		Percentage:  0.0,
		Mark:        naMark,
		ClassSuffix: "na",
	}

	if all == 0 {
		return g
	}

	g.Percentage = value / float32(all)
	if g.Percentage <= 1 {
		g.Percentage *= 100
	}

	switch {
	case g.Percentage >= 97.0:
		g.Mark = "A+"
		g.ClassSuffix = "a"
	case g.Percentage >= 93.0:
		g.Mark = "A"
		g.ClassSuffix = "a"
	case g.Percentage >= 90.0:
		g.Mark = "A-"
		g.ClassSuffix = "a"
	case g.Percentage >= 87.0:
		g.Mark = "B+"
		g.ClassSuffix = "b"
	case g.Percentage >= 83.0:
		g.Mark = "B"
		g.ClassSuffix = "b"
	case g.Percentage >= 80.0:
		g.Mark = "B-"
		g.ClassSuffix = "b"
	case g.Percentage >= 77.0:
		g.Mark = "C+"
		g.ClassSuffix = "c"
	case g.Percentage >= 73.0:
		g.Mark = "C"
		g.ClassSuffix = "c"
	case g.Percentage >= 70.0:
		g.Mark = "C-"
		g.ClassSuffix = "c"
	case g.Percentage >= 67.0:
		g.Mark = "D+"
		g.ClassSuffix = "d"
	case g.Percentage >= 63.0:
		g.Mark = "D"
		g.ClassSuffix = "d"
	case g.Percentage >= 60.0:
		g.Mark = "D-"
		g.ClassSuffix = "d"
	case g.Percentage < 60.0:
		g.Mark = "F"
		g.ClassSuffix = "f"
	}

	return g
}

func MapKeysToString(m map[string]interface{}, sep string) string {
	var keysList []string

	for k := range m {
		keysList = append(keysList, k)
	}

	return strings.Join(keysList, sep)
}

func exportFullReportToHtml(
	s *db.Statistics, reportTime time.Time, wafName string,
	url string, openApiFile string, args string, ignoreUnresolved bool,
) (fileName string, err error) {
	data := htmlReport{
		IgnoreUnresolved: ignoreUnresolved,
		WafName:          wafName,
		Url:              url,
		WafTestingDate:   reportTime.Format("02 January 2006"),
		GtwVersion:       version.Version,
		TestCasesFP:      s.TestCasesFingerprint,
		OpenApiFile:      openApiFile,
		Args:             args,
		ComparisonTable: []*comparisonTableRow{
			{
				Name:         "ModSecurity PARANOIA=1",
				ApiSec:       computeGrade(42.9, 1),
				AppSec:       computeGrade(30.5, 1),
				OverallScore: computeGrade(36.7, 1),
			},
			{
				Name:         "ModSecurity PARANOIA=2",
				ApiSec:       computeGrade(78.6, 1),
				AppSec:       computeGrade(34.8, 1),
				OverallScore: computeGrade(56.7, 1),
			},
			{
				Name:         "ModSecurity PARANOIA=3",
				ApiSec:       computeGrade(92.9, 1),
				AppSec:       computeGrade(38.3, 1),
				OverallScore: computeGrade(65.6, 1),
			},
			{
				Name:         "ModSecurity PARANOIA=4",
				ApiSec:       computeGrade(100, 1),
				AppSec:       computeGrade(40.8, 1),
				OverallScore: computeGrade(70.4, 1),
			},
		},
	}

	var apiSecNegBlockedNum int
	var apiSecNegNum int
	var appSecNegBlockedNum int
	var appSecNegNum int

	for _, test := range s.NegativeTests.Blocked {
		if isApiTest(test.TestSet) {
			apiSecNegNum++
			apiSecNegBlockedNum++
		} else {
			appSecNegNum++
			appSecNegBlockedNum++
		}
	}
	for _, test := range s.NegativeTests.Bypasses {
		if isApiTest(test.TestSet) {
			apiSecNegNum++
		} else {
			appSecNegNum++
		}
	}

	var apiSecPosBypassNum int
	var apiSecPosNum int
	var appSecPosBypassNum int
	var appSecPosNum int

	for _, test := range s.PositiveTests.TruePositive {
		if isApiTest(test.TestSet) {
			apiSecPosNum++
			apiSecPosBypassNum++
		} else {
			appSecPosNum++
			appSecPosBypassNum++
		}
	}
	for _, test := range s.PositiveTests.FalsePositive {
		if isApiTest(test.TestSet) {
			apiSecPosNum++
		} else {
			appSecPosNum++
		}
	}

	divider := 0
	data.ApiSec.TrueNegative = computeGrade(float32(apiSecNegBlockedNum), apiSecNegNum)
	data.ApiSec.TruePositive = computeGrade(float32(apiSecPosBypassNum), apiSecPosNum)
	if data.ApiSec.TrueNegative.Mark != naMark {
		divider++
	}
	if data.ApiSec.TruePositive.Mark != naMark {
		divider++
	}
	data.ApiSec.Grade = computeGrade(
		data.ApiSec.TrueNegative.Percentage+
			data.ApiSec.TruePositive.Percentage,
		divider,
	)

	divider = 0

	data.AppSec.TrueNegative = computeGrade(float32(appSecNegBlockedNum), appSecNegNum)
	data.AppSec.TruePositive = computeGrade(float32(appSecPosBypassNum), appSecPosNum)
	if data.AppSec.TrueNegative.Mark != naMark {
		divider++
	}
	if data.AppSec.TruePositive.Mark != naMark {
		divider++
	}
	data.AppSec.Grade = computeGrade(
		data.AppSec.TrueNegative.Percentage+
			data.AppSec.TruePositive.Percentage,
		divider,
	)

	divider = 0
	if data.ApiSec.Grade.Mark != naMark {
		divider++
	}
	if data.AppSec.Grade.Mark != naMark {
		divider++
	}
	data.Overall = computeGrade(
		data.ApiSec.Grade.Percentage+data.AppSec.Grade.Percentage, divider)

	apiChart, appChart, err := generateCharts(s)
	if err != nil {
		return "", errors.Wrap(err, "couldn't generate chart scripts")
	}

	if apiChart != nil {
		v := template.HTML(*apiChart)
		data.ApiChartScript = &v
	}
	if appChart != nil {
		v := template.HTML(*appChart)
		data.AppChartScript = &v
	}

	data.NegativeTests.SummaryTable = make(map[string]*testSetSummary)
	for _, row := range s.NegativeTests.SummaryTable {
		if _, ok := data.NegativeTests.SummaryTable[row.TestSet]; !ok {
			data.NegativeTests.SummaryTable[row.TestSet] = &testSetSummary{}
		}

		testSetSum := data.NegativeTests.SummaryTable[row.TestSet]

		testSetSum.TestCases = append(testSetSum.TestCases, row)

		testSetSum.Sent += row.Sent
		testSetSum.Blocked += row.Blocked
		testSetSum.Bypassed += row.Bypassed
		testSetSum.Unresolved += row.Unresolved
		testSetSum.Failed += row.Failed

		if row.Blocked+row.Bypassed != 0 {
			testSetSum.resolvedTestCasesNumber += 1
			testSetSum.Percentage += row.Percentage
		}
	}
	for _, testSetSum := range data.NegativeTests.SummaryTable {
		testSetSum.Percentage = db.Round(testSetSum.Percentage / float64(testSetSum.resolvedTestCasesNumber))
	}

	data.PositiveTests.SummaryTable = make(map[string]*testSetSummary)
	for _, row := range s.PositiveTests.SummaryTable {
		if _, ok := data.PositiveTests.SummaryTable[row.TestSet]; !ok {
			data.PositiveTests.SummaryTable[row.TestSet] = &testSetSummary{}
		}

		testSetSum := data.PositiveTests.SummaryTable[row.TestSet]

		testSetSum.TestCases = append(testSetSum.TestCases, row)

		testSetSum.Sent += row.Sent
		testSetSum.Blocked += row.Blocked
		testSetSum.Bypassed += row.Bypassed
		testSetSum.Unresolved += row.Unresolved
		testSetSum.Failed += row.Failed

		if row.Blocked+row.Bypassed != 0 {
			testSetSum.resolvedTestCasesNumber += 1
			testSetSum.Percentage += row.Percentage
		}
	}
	for _, testSetSum := range data.PositiveTests.SummaryTable {
		testSetSum.Percentage = db.Round(testSetSum.Percentage / float64(testSetSum.resolvedTestCasesNumber))
	}

	// map[paths]map[payload]map[statusCode]*testDetails
	negBypassed := make(map[string]map[string]map[int]*testDetails)
	for _, d := range s.NegativeTests.Bypasses {
		paths := strings.Join(d.AdditionalInfo, "\n")

		if _, ok := negBypassed[paths]; !ok {
			// map[payload]map[statusCode]*testDetails
			negBypassed[paths] = make(map[string]map[int]*testDetails)
		}

		if _, ok := negBypassed[paths][d.Payload]; !ok {
			// map[statusCode]*testDetails
			negBypassed[paths][d.Payload] = make(map[int]*testDetails)
		}

		if _, ok := negBypassed[paths][d.Payload][d.ResponseStatusCode]; !ok {
			negBypassed[paths][d.Payload][d.ResponseStatusCode] = &testDetails{
				Encoders:     make(map[string]any),
				Placeholders: make(map[string]any),
			}
		}

		negBypassed[paths][d.Payload][d.ResponseStatusCode].TestCase = d.TestCase
		negBypassed[paths][d.Payload][d.ResponseStatusCode].Encoders[d.Encoder] = nil
		negBypassed[paths][d.Payload][d.ResponseStatusCode].Placeholders[d.Placeholder] = nil
	}

	// map[payload]map[statusCode]*testDetails
	negUnresolved := make(map[string]map[int]*testDetails)
	for _, d := range s.NegativeTests.Unresolved {
		if _, ok := negUnresolved[d.Payload]; !ok {
			// map[statusCode]*testDetails
			negUnresolved[d.Payload] = make(map[int]*testDetails)
		}

		if _, ok := negUnresolved[d.Payload][d.ResponseStatusCode]; !ok {
			negUnresolved[d.Payload][d.ResponseStatusCode] = &testDetails{
				Encoders:     make(map[string]any),
				Placeholders: make(map[string]any),
			}
		}

		negUnresolved[d.Payload][d.ResponseStatusCode].TestCase = d.TestCase
		negUnresolved[d.Payload][d.ResponseStatusCode].Encoders[d.Encoder] = nil
		negUnresolved[d.Payload][d.ResponseStatusCode].Placeholders[d.Placeholder] = nil
	}

	// map[payload]map[statusCode]*testDetails
	posBlocked := make(map[string]map[int]*testDetails)
	for _, d := range s.PositiveTests.FalsePositive {
		if _, ok := posBlocked[d.Payload]; !ok {
			// map[statusCode]*testDetails
			posBlocked[d.Payload] = make(map[int]*testDetails)
		}

		if _, ok := posBlocked[d.Payload][d.ResponseStatusCode]; !ok {
			posBlocked[d.Payload][d.ResponseStatusCode] = &testDetails{
				Encoders:     make(map[string]any),
				Placeholders: make(map[string]any),
			}
		}

		posBlocked[d.Payload][d.ResponseStatusCode].TestCase = d.TestCase
		posBlocked[d.Payload][d.ResponseStatusCode].Encoders[d.Encoder] = nil
		posBlocked[d.Payload][d.ResponseStatusCode].Placeholders[d.Placeholder] = nil
	}

	// map[payload]map[statusCode]*testDetails
	posBypassed := make(map[string]map[int]*testDetails)
	for _, d := range s.PositiveTests.TruePositive {
		if _, ok := posBypassed[d.Payload]; !ok {
			// map[statusCode]*testDetails
			posBypassed[d.Payload] = make(map[int]*testDetails)
		}

		if _, ok := posBypassed[d.Payload][d.ResponseStatusCode]; !ok {
			posBypassed[d.Payload][d.ResponseStatusCode] = &testDetails{
				Encoders:     make(map[string]any),
				Placeholders: make(map[string]any),
			}
		}

		posBypassed[d.Payload][d.ResponseStatusCode].TestCase = d.TestCase
		posBypassed[d.Payload][d.ResponseStatusCode].Encoders[d.Encoder] = nil
		posBypassed[d.Payload][d.ResponseStatusCode].Placeholders[d.Placeholder] = nil
	}

	// map[payload]map[statusCode]*testDetails
	posUnresolved := make(map[string]map[int]*testDetails)
	for _, d := range s.PositiveTests.Unresolved {
		if _, ok := posUnresolved[d.Payload]; !ok {
			// map[statusCode]*testDetails
			posUnresolved[d.Payload] = make(map[int]*testDetails)
		}

		if _, ok := posUnresolved[d.Payload][d.ResponseStatusCode]; !ok {
			posUnresolved[d.Payload][d.ResponseStatusCode] = &testDetails{
				Encoders:     make(map[string]any),
				Placeholders: make(map[string]any),
			}
		}

		posUnresolved[d.Payload][d.ResponseStatusCode].TestCase = d.TestCase
		posUnresolved[d.Payload][d.ResponseStatusCode].Encoders[d.Encoder] = nil
		posUnresolved[d.Payload][d.ResponseStatusCode].Placeholders[d.Placeholder] = nil
	}

	data.ScannedPaths = s.Paths

	data.NegativeTests.Bypassed = negBypassed
	data.NegativeTests.Unresolved = negUnresolved
	data.NegativeTests.Failed = s.NegativeTests.Failed
	data.NegativeTests.Percentage = s.WafScore
	data.NegativeTests.TotalSent = s.NegativeTests.AllRequestsNumber
	data.NegativeTests.BlockedRequestsNumber = s.NegativeTests.BlockedRequestsNumber
	data.NegativeTests.BypassedRequestsNumber = s.NegativeTests.BypassedRequestsNumber
	data.NegativeTests.UnresolvedRequestsNumber = s.NegativeTests.UnresolvedRequestsNumber
	data.NegativeTests.FailedRequestsNumber = s.NegativeTests.FailedRequestsNumber

	data.PositiveTests.Blocked = posBlocked
	data.PositiveTests.Bypassed = posBypassed
	data.PositiveTests.Unresolved = posUnresolved
	data.PositiveTests.Failed = s.PositiveTests.Failed
	data.PositiveTests.Percentage = s.PositiveTests.ResolvedTrueRequestsPercentage
	data.PositiveTests.TotalSent = s.PositiveTests.AllRequestsNumber
	data.PositiveTests.BlockedRequestsNumber = s.PositiveTests.BlockedRequestsNumber
	data.PositiveTests.BypassedRequestsNumber = s.PositiveTests.BypassedRequestsNumber
	data.PositiveTests.UnresolvedRequestsNumber = s.PositiveTests.UnresolvedRequestsNumber
	data.PositiveTests.FailedRequestsNumber = s.PositiveTests.FailedRequestsNumber

	data.TotalSent = data.NegativeTests.TotalSent + data.PositiveTests.TotalSent
	data.BlockedRequestsNumber = data.NegativeTests.BlockedRequestsNumber + data.PositiveTests.BlockedRequestsNumber
	data.BypassedRequestsNumber = data.NegativeTests.BypassedRequestsNumber + data.PositiveTests.BypassedRequestsNumber
	data.UnresolvedRequestsNumber = data.NegativeTests.UnresolvedRequestsNumber + data.PositiveTests.UnresolvedRequestsNumber
	data.FailedRequestsNumber = data.NegativeTests.FailedRequestsNumber + data.PositiveTests.FailedRequestsNumber

	templ := template.Must(
		template.New("report").
			Funcs(template.FuncMap{
				"script": func(s string) template.HTML {
					return template.HTML(s)
				},
			}).
			Funcs(template.FuncMap{
				"StringsJoin":     strings.Join,
				"StringsSplit":    strings.Split,
				"MapKeysToString": MapKeysToString,
			}).Parse(htmlTemplate))

	var buffer bytes.Buffer

	err = templ.Execute(io.MultiWriter(&buffer), data)
	if err != nil {
		return "", errors.Wrap(err, "couldn't execute template")
	}

	file, err := os.CreateTemp("", "gotestwaf_report_*.html")
	if err != nil {
		return "", errors.Wrap(err, "couldn't create a temporary file")
	}
	defer file.Close()

	fileName = file.Name()

	file.Write(buffer.Bytes())

	err = os.Chmod(fileName, 0644)

	return fileName, err
}
