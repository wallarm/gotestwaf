package report

import (
	"bytes"
	_ "embed"
	"html/template"
	"io"
	"os"
	"strings"
	"time"

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

type reportInfo struct {
	IgnoreUnresolved bool

	WafName        string
	Url            string
	WafTestingDate string
	GtwVersion     string

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

	ComparisonTable []comparisonTableRow

	SummaryTable []db.SummaryTableRow

	NegativeTests struct {
		Blocked    []db.TestDetails
		Bypassed   []db.TestDetails
		Unresolved []db.TestDetails
		Failed     []db.FailedDetails

		BlockedRequestsNumber    int
		BypassedRequestsNumber   int
		UnresolvedRequestsNumber int
		FailedRequestsNumber     int
	}

	PositiveTests struct {
		Blocked    []db.TestDetails
		Bypassed   []db.TestDetails
		Unresolved []db.TestDetails
		Failed     []db.FailedDetails

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

func ExportToPDF(
	s *db.Statistics, reportFile string, reportTime time.Time,
	wafName string, url string, ignoreUnresolved bool, toHTML bool,
) error {
	data := reportInfo{
		IgnoreUnresolved: ignoreUnresolved,
		WafName:          wafName,
		Url:              url,
		WafTestingDate:   reportTime.Format("02 January 2006"),
		GtwVersion:       version.Version,
		SummaryTable:     append(s.SummaryTable, s.PositiveTests.SummaryTable...),
		ComparisonTable: []comparisonTableRow{
			{
				Name:         "ModSecurity PARANOIA=1",
				ApiSec:       computeGrade(42.9, 1),
				AppSec:       computeGrade(30.3, 1),
				OverallScore: computeGrade(36.6, 1),
			},
			{
				Name:         "ModSecurity PARANOIA=2",
				ApiSec:       computeGrade(78.6, 1),
				AppSec:       computeGrade(34.7, 1),
				OverallScore: computeGrade(56.6, 1),
			},
			{
				Name:         "ModSecurity PARANOIA=3",
				ApiSec:       computeGrade(92.9, 1),
				AppSec:       computeGrade(39.4, 1),
				OverallScore: computeGrade(66.2, 1),
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

	for _, test := range s.Blocked {
		if isApiTest(test.TestSet) {
			apiSecNegNum++
			apiSecNegBlockedNum++
		} else {
			appSecNegNum++
			appSecNegBlockedNum++
		}
	}
	for _, test := range s.Bypasses {
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
		return err
	}

	if apiChart != nil {
		v := template.HTML(*apiChart)
		data.ApiChartScript = &v
	}
	if appChart != nil {
		v := template.HTML(*appChart)
		data.AppChartScript = &v
	}

	data.NegativeTests.Blocked = s.Blocked
	data.NegativeTests.Bypassed = s.Bypasses
	data.NegativeTests.Unresolved = s.Unresolved
	data.NegativeTests.Failed = s.Failed
	data.NegativeTests.BlockedRequestsNumber = s.BlockedRequestsNumber
	data.NegativeTests.BypassedRequestsNumber = s.BypassedRequestsNumber
	data.NegativeTests.UnresolvedRequestsNumber = s.UnresolvedRequestsNumber
	data.NegativeTests.FailedRequestsNumber = s.FailedRequestsNumber

	data.PositiveTests.Blocked = s.PositiveTests.FalsePositive
	data.PositiveTests.Bypassed = s.PositiveTests.TruePositive
	data.PositiveTests.Unresolved = s.PositiveTests.Unresolved
	data.PositiveTests.Failed = s.PositiveTests.Failed
	data.PositiveTests.BlockedRequestsNumber = s.PositiveTests.BlockedRequestsNumber
	data.PositiveTests.BypassedRequestsNumber = s.PositiveTests.BypassedRequestsNumber
	data.PositiveTests.UnresolvedRequestsNumber = s.PositiveTests.UnresolvedRequestsNumber
	data.PositiveTests.FailedRequestsNumber = s.PositiveTests.FailedRequestsNumber

	templ := template.Must(template.New("report").Funcs(template.FuncMap{
		"script": func(b []byte) template.HTML {
			return template.HTML(b)
		},
	}).Parse(htmlTemplate))

	var buffer bytes.Buffer

	err = templ.Execute(io.MultiWriter(&buffer), data)
	if err != nil {
		return err
	}

	if toHTML {
		report, err := os.Create(reportFile)
		if err != nil {
			return err
		}
		defer report.Close()

		_, err = report.Write(buffer.Bytes())
		if err != nil {
			return err
		}
	} else {
		err = renderToPDF(buffer.Bytes(), reportFile)
		if err != nil {
			return err
		}
	}

	err = os.Chmod(reportFile, 0644)

	return err
}
