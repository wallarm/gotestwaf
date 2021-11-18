package report

import (
	"bytes"
	_ "embed"
	"html/template"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"

	"github.com/wallarm/gotestwaf/internal/db"
	"github.com/wallarm/gotestwaf/internal/version"
)

const naMark = "N/A"

//go:embed report_template.html
var htmlTemplate string

type grade struct {
	Percentage float32
	Mark       string
	Color      string
}

type reportInfo struct {
	IgnoreUnresolved bool

	WafName        string
	Url            string
	WafTestingDate string
	GtwVersion     string

	ChartScript template.HTML

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

	ComparisonTable []struct {
		ApiSec       grade
		AppSec       grade
		OverallScore grade
	}

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
		Percentage: 0.0,
		Mark:       naMark,
		Color:      gray,
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
		g.Color = green
	case g.Percentage >= 93.0:
		g.Mark = "A"
		g.Color = green
	case g.Percentage >= 90.0:
		g.Mark = "A-"
		g.Color = green
	case g.Percentage >= 87.0:
		g.Mark = "B+"
		g.Color = lightGreen
	case g.Percentage >= 83.0:
		g.Mark = "B"
		g.Color = lightGreen
	case g.Percentage >= 80.0:
		g.Mark = "B-"
		g.Color = lightGreen
	case g.Percentage >= 77.0:
		g.Mark = "C+"
		g.Color = yellow
	case g.Percentage >= 73.0:
		g.Mark = "C"
		g.Color = yellow
	case g.Percentage >= 70.0:
		g.Mark = "C-"
		g.Color = yellow
	case g.Percentage >= 67.0:
		g.Mark = "D+"
		g.Color = orange
	case g.Percentage >= 63.0:
		g.Mark = "D"
		g.Color = orange
	case g.Percentage >= 60.0:
		g.Mark = "D-"
		g.Color = orange
	case g.Percentage < 60.0:
		g.Mark = "F"
		g.Color = red
	}

	return g
}

func ExportToPDF(s *db.Statistics, reportFile string, reportTime time.Time, wafName string, url string, ignoreUnresolved bool) error {
	data := reportInfo{
		IgnoreUnresolved: ignoreUnresolved,
		WafName:          wafName,
		Url:              url,
		WafTestingDate:   reportTime.Format("02 January 2006"),
		GtwVersion:       version.Version,
		SummaryTable:     append(s.SummaryTable, s.PositiveTests.SummaryTable...),
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

	sumTable := append(s.SummaryTable, s.PositiveTests.SummaryTable...)
	script, err := generateChartScript(sumTable)
	if err != nil {
		return err
	}

	data.ChartScript = template.HTML(*script)

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

	// TODO: delete
	f1, err := os.Create("report.html")
	if err != nil {
		return err
	}

	var buffer bytes.Buffer

	err = templ.Execute(io.MultiWriter(&buffer), data)
	if err != nil {
		return err
	}

	// TODO: delete
	_, err = f1.Write(buffer.Bytes())
	if err != nil {
		return err
	}

	// Create new PDF generator
	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		log.Fatal(err)
	}
	pdfg.PageSize.Set(wkhtmltopdf.PageSizeA4)
	pdfg.Dpi.Set(192)

	page := wkhtmltopdf.NewPageReader(&buffer)
	page.DebugJavascript.Set(true)
	page.NoStopSlowScripts.Set(true)

	// Add a new page
	pdfg.AddPage(page)

	// Create PDF document in internal buffer
	err = pdfg.Create()
	if err != nil {
		return err
	}

	// Write buffer contents to file on disk
	err = pdfg.WriteFile(reportFile)
	if err != nil {
		return err
	}

	return nil
}
