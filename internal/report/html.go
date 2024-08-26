package report

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/db"
	"github.com/wallarm/gotestwaf/internal/helpers"
	"github.com/wallarm/gotestwaf/internal/version"
	"github.com/wallarm/gotestwaf/pkg/report"
)

const (
	naMark = "N/A"

	maxUntruncatedPayloadLength = 1100
	truncatedPartsLength        = 150
)

var (
	prepareHTMLReportOnce sync.Once
	htmlReportData        *report.HtmlReport

	comparisonTable = []*report.ComparisonTableRow{
		{
			Name:         "ModSecurity PARANOIA=1",
			ApiSec:       computeGrade(42.86, 1),
			AppSec:       computeGrade(67.57, 1),
			OverallScore: computeGrade(55.22, 1),
		},
		{
			Name:         "ModSecurity PARANOIA=2",
			ApiSec:       computeGrade(57.14, 1),
			AppSec:       computeGrade(58.94, 1),
			OverallScore: computeGrade(58.04, 1),
		},
		{
			Name:         "ModSecurity PARANOIA=3",
			ApiSec:       computeGrade(85.71, 1),
			AppSec:       computeGrade(50.86, 1),
			OverallScore: computeGrade(68.29, 1),
		},
		{
			Name:         "ModSecurity PARANOIA=4",
			ApiSec:       computeGrade(100.00, 1),
			AppSec:       computeGrade(36.76, 1),
			OverallScore: computeGrade(68.38, 1),
		},
	}

	wallarmResult = &report.ComparisonTableRow{
		Name:         "Wallarm",
		ApiSec:       computeGrade(100, 1),
		AppSec:       computeGrade(97.74, 1),
		OverallScore: computeGrade(98.87, 1),
	}
)

func getGrade(grade float64, na bool) *report.Grade {
	g := &report.Grade{
		Percentage:     0.0,
		Mark:           naMark,
		CSSClassSuffix: "na",
	}

	if na {
		return g
	}

	g.Percentage = grade
	if g.Percentage <= 1 {
		g.Percentage *= 100
	}

	switch {
	case g.Percentage >= 97.0:
		g.Mark = "A+"
		g.CSSClassSuffix = "a"
	case g.Percentage >= 93.0:
		g.Mark = "A"
		g.CSSClassSuffix = "a"
	case g.Percentage >= 90.0:
		g.Mark = "A-"
		g.CSSClassSuffix = "a"
	case g.Percentage >= 87.0:
		g.Mark = "B+"
		g.CSSClassSuffix = "b"
	case g.Percentage >= 83.0:
		g.Mark = "B"
		g.CSSClassSuffix = "b"
	case g.Percentage >= 80.0:
		g.Mark = "B-"
		g.CSSClassSuffix = "b"
	case g.Percentage >= 77.0:
		g.Mark = "C+"
		g.CSSClassSuffix = "c"
	case g.Percentage >= 73.0:
		g.Mark = "C"
		g.CSSClassSuffix = "c"
	case g.Percentage >= 70.0:
		g.Mark = "C-"
		g.CSSClassSuffix = "c"
	case g.Percentage >= 67.0:
		g.Mark = "D+"
		g.CSSClassSuffix = "d"
	case g.Percentage >= 63.0:
		g.Mark = "D"
		g.CSSClassSuffix = "d"
	case g.Percentage >= 60.0:
		g.Mark = "D-"
		g.CSSClassSuffix = "d"
	case g.Percentage < 60.0:
		g.Mark = "F"
		g.CSSClassSuffix = "f"
	}

	return g
}

func computeGrade(value float64, all int) *report.Grade {
	if all == 0 {
		return getGrade(0.0, true)
	}

	return getGrade(value/float64(all), false)
}

// truncatePayload replaces the middle part of the payload if
// it is longer than maxUntruncatedPayloadLength.
func truncatePayload(payload string) string {
	payloadLength := len(payload)

	if payloadLength <= maxUntruncatedPayloadLength {
		return payload
	}

	truncatedLength := payloadLength - 2*truncatedPartsLength

	truncatedPayload := fmt.Sprintf(
		"%s … truncated %d symbols … %s",
		payload[:truncatedPartsLength],
		truncatedLength,
		payload[payloadLength-truncatedPartsLength:],
	)

	return truncatedPayload
}

// prepareHTMLFullReport prepares ready data to insert into the HTML template.
func prepareHTMLFullReport(
	s *db.Statistics, reportTime time.Time, wafName string,
	url string, openApiFile string, args []string, ignoreUnresolved bool, includePayloads bool,
) (*report.HtmlReport, error) {
	data := &report.HtmlReport{
		IgnoreUnresolved: ignoreUnresolved,
		IncludePayloads:  includePayloads,
		WafName:          wafName,
		Url:              url,
		WafTestingDate:   reportTime.Format("02 January 2006"),
		GtwVersion:       version.Version,
		TestCasesFP:      s.TestCasesFingerprint,
		OpenApiFile:      openApiFile,
		Args:             args,
		ComparisonTable:  comparisonTable,
		WallarmResult:    wallarmResult,
	}

	if s.Score.ApiSec.TruePositive < 0 {
		data.ApiSec.TruePositiveTestsGrade = getGrade(0.0, true)
	} else {
		data.ApiSec.TruePositiveTestsGrade = getGrade(s.Score.ApiSec.TruePositive, false)
	}

	if s.Score.ApiSec.TrueNegative < 0 {
		data.ApiSec.TrueNegativeTestsGrade = getGrade(0.0, true)
	} else {
		data.ApiSec.TrueNegativeTestsGrade = getGrade(s.Score.ApiSec.TrueNegative, false)
	}

	if s.Score.ApiSec.Average < 0 {
		data.ApiSec.Grade = getGrade(0.0, true)
	} else {
		data.ApiSec.Grade = getGrade(s.Score.ApiSec.Average, false)
	}

	if s.Score.AppSec.TruePositive < 0 {
		data.AppSec.TruePositiveTestsGrade = getGrade(0.0, true)
	} else {
		data.AppSec.TruePositiveTestsGrade = getGrade(s.Score.AppSec.TruePositive, false)
	}

	if s.Score.AppSec.TrueNegative < 0 {
		data.AppSec.TrueNegativeTestsGrade = getGrade(0.0, true)
	} else {
		data.AppSec.TrueNegativeTestsGrade = getGrade(s.Score.AppSec.TrueNegative, false)
	}

	if s.Score.AppSec.Average < 0 {
		data.AppSec.Grade = getGrade(0.0, true)
	} else {
		data.AppSec.Grade = getGrade(s.Score.AppSec.Average, false)
	}

	data.Overall = getGrade(s.Score.Average, false)

	apiIndicators, apiItems, appIndicators, appItems := generateChartData(s)

	data.ApiSecChartData.Indicators = apiIndicators
	data.ApiSecChartData.Items = apiItems
	data.AppSecChartData.Indicators = appIndicators
	data.AppSecChartData.Items = appItems

	data.TruePositiveTests.SummaryTable = make(map[string]*report.TestSetSummary)
	for _, row := range s.TruePositiveTests.SummaryTable {
		if _, ok := data.TruePositiveTests.SummaryTable[row.TestSet]; !ok {
			data.TruePositiveTests.SummaryTable[row.TestSet] = &report.TestSetSummary{}
		}

		testSetSum := data.TruePositiveTests.SummaryTable[row.TestSet]

		testSetSum.TestCases = append(testSetSum.TestCases, row)

		testSetSum.Sent += row.Sent
		testSetSum.Blocked += row.Blocked
		testSetSum.Bypassed += row.Bypassed
		testSetSum.Unresolved += row.Unresolved
		testSetSum.Failed += row.Failed

		if row.Blocked+row.Bypassed != 0 {
			testSetSum.ResolvedTestCasesNumber += 1
			testSetSum.Percentage += row.Percentage
		}
	}
	for _, testSetSum := range data.TruePositiveTests.SummaryTable {
		testSetSum.Percentage = db.Round(testSetSum.Percentage / float64(testSetSum.ResolvedTestCasesNumber))
	}

	data.TrueNegativeTests.SummaryTable = make(map[string]*report.TestSetSummary)
	for _, row := range s.TrueNegativeTests.SummaryTable {
		if _, ok := data.TrueNegativeTests.SummaryTable[row.TestSet]; !ok {
			data.TrueNegativeTests.SummaryTable[row.TestSet] = &report.TestSetSummary{}
		}

		testSetSum := data.TrueNegativeTests.SummaryTable[row.TestSet]

		testSetSum.TestCases = append(testSetSum.TestCases, row)

		testSetSum.Sent += row.Sent
		testSetSum.Blocked += row.Blocked
		testSetSum.Bypassed += row.Bypassed
		testSetSum.Unresolved += row.Unresolved
		testSetSum.Failed += row.Failed

		if row.Blocked+row.Bypassed != 0 {
			testSetSum.ResolvedTestCasesNumber += 1
			testSetSum.Percentage += row.Percentage
		}
	}
	for _, testSetSum := range data.TrueNegativeTests.SummaryTable {
		testSetSum.Percentage = db.Round(testSetSum.Percentage / float64(testSetSum.ResolvedTestCasesNumber))
	}

	if includePayloads {
		// map[paths]map[payload]map[statusCode]*testDetails
		negBypassed := make(map[string]map[string]map[int]*report.TestDetails)
		for _, d := range s.TruePositiveTests.Bypasses {
			paths := strings.Join(d.AdditionalInfo, "\n")

			if _, ok := negBypassed[paths]; !ok {
				// map[payload]map[statusCode]*testDetails
				negBypassed[paths] = make(map[string]map[int]*report.TestDetails)
			}

			payload := truncatePayload(d.Payload)

			if _, ok := negBypassed[paths][payload]; !ok {
				// map[statusCode]*testDetails
				negBypassed[paths][payload] = make(map[int]*report.TestDetails)
			}

			if _, ok := negBypassed[paths][payload][d.ResponseStatusCode]; !ok {
				negBypassed[paths][payload][d.ResponseStatusCode] = &report.TestDetails{
					Encoders:     make(map[string]any),
					Placeholders: make(map[string]any),
				}
			}

			negBypassed[paths][payload][d.ResponseStatusCode].TestCase = d.TestCase
			negBypassed[paths][payload][d.ResponseStatusCode].Encoders[d.Encoder] = nil
			negBypassed[paths][payload][d.ResponseStatusCode].Placeholders[d.Placeholder] = nil
		}

		// map[payload]map[statusCode]*testDetails
		negUnresolved := make(map[string]map[int]*report.TestDetails)
		for _, d := range s.TruePositiveTests.Unresolved {
			payload := truncatePayload(d.Payload)

			if _, ok := negUnresolved[payload]; !ok {
				// map[statusCode]*testDetails
				negUnresolved[payload] = make(map[int]*report.TestDetails)
			}

			if _, ok := negUnresolved[payload][d.ResponseStatusCode]; !ok {
				negUnresolved[payload][d.ResponseStatusCode] = &report.TestDetails{
					Encoders:     make(map[string]any),
					Placeholders: make(map[string]any),
				}
			}

			negUnresolved[payload][d.ResponseStatusCode].TestCase = d.TestCase
			negUnresolved[payload][d.ResponseStatusCode].Encoders[d.Encoder] = nil
			negUnresolved[payload][d.ResponseStatusCode].Placeholders[d.Placeholder] = nil
		}

		data.TruePositiveTests.Bypassed = negBypassed
		data.TruePositiveTests.Unresolved = negUnresolved
		data.TruePositiveTests.Failed = s.TruePositiveTests.Failed

		// map[payload]map[statusCode]*testDetails
		posBlocked := make(map[string]map[int]*report.TestDetails)
		for _, d := range s.TrueNegativeTests.Blocked {
			payload := truncatePayload(d.Payload)

			if _, ok := posBlocked[payload]; !ok {
				// map[statusCode]*testDetails
				posBlocked[payload] = make(map[int]*report.TestDetails)
			}

			if _, ok := posBlocked[payload][d.ResponseStatusCode]; !ok {
				posBlocked[payload][d.ResponseStatusCode] = &report.TestDetails{
					Encoders:     make(map[string]any),
					Placeholders: make(map[string]any),
				}
			}

			posBlocked[payload][d.ResponseStatusCode].TestCase = d.TestCase
			posBlocked[payload][d.ResponseStatusCode].Encoders[d.Encoder] = nil
			posBlocked[payload][d.ResponseStatusCode].Placeholders[d.Placeholder] = nil
		}

		// map[payload]map[statusCode]*testDetails
		posBypassed := make(map[string]map[int]*report.TestDetails)
		for _, d := range s.TrueNegativeTests.Bypasses {
			payload := truncatePayload(d.Payload)

			if _, ok := posBypassed[payload]; !ok {
				// map[statusCode]*testDetails
				posBypassed[payload] = make(map[int]*report.TestDetails)
			}

			if _, ok := posBypassed[payload][d.ResponseStatusCode]; !ok {
				posBypassed[payload][d.ResponseStatusCode] = &report.TestDetails{
					Encoders:     make(map[string]any),
					Placeholders: make(map[string]any),
				}
			}

			posBypassed[payload][d.ResponseStatusCode].TestCase = d.TestCase
			posBypassed[payload][d.ResponseStatusCode].Encoders[d.Encoder] = nil
			posBypassed[payload][d.ResponseStatusCode].Placeholders[d.Placeholder] = nil
		}

		// map[payload]map[statusCode]*testDetails
		posUnresolved := make(map[string]map[int]*report.TestDetails)
		for _, d := range s.TrueNegativeTests.Unresolved {
			payload := truncatePayload(d.Payload)

			if _, ok := posUnresolved[payload]; !ok {
				// map[statusCode]*testDetails
				posUnresolved[payload] = make(map[int]*report.TestDetails)
			}

			if _, ok := posUnresolved[payload][d.ResponseStatusCode]; !ok {
				posUnresolved[payload][d.ResponseStatusCode] = &report.TestDetails{
					Encoders:     make(map[string]any),
					Placeholders: make(map[string]any),
				}
			}

			posUnresolved[payload][d.ResponseStatusCode].TestCase = d.TestCase
			posUnresolved[payload][d.ResponseStatusCode].Encoders[d.Encoder] = nil
			posUnresolved[payload][d.ResponseStatusCode].Placeholders[d.Placeholder] = nil
		}

		data.TrueNegativeTests.Blocked = posBlocked
		data.TrueNegativeTests.Bypassed = posBypassed
		data.TrueNegativeTests.Unresolved = posUnresolved
		data.TrueNegativeTests.Failed = s.TrueNegativeTests.Failed
	}

	data.ScannedPaths = s.Paths

	data.TruePositiveTests.Percentage = s.TruePositiveTests.ResolvedBlockedRequestsPercentage
	data.TruePositiveTests.TotalSent = s.TruePositiveTests.ReqStats.AllRequestsNumber
	data.TruePositiveTests.BlockedRequestsNumber = s.TruePositiveTests.ReqStats.BlockedRequestsNumber
	data.TruePositiveTests.BypassedRequestsNumber = s.TruePositiveTests.ReqStats.BypassedRequestsNumber
	data.TruePositiveTests.UnresolvedRequestsNumber = s.TruePositiveTests.ReqStats.UnresolvedRequestsNumber
	data.TruePositiveTests.FailedRequestsNumber = s.TruePositiveTests.ReqStats.FailedRequestsNumber

	data.TrueNegativeTests.Percentage = s.TrueNegativeTests.ResolvedBypassedRequestsPercentage
	data.TrueNegativeTests.TotalSent = s.TrueNegativeTests.ReqStats.AllRequestsNumber
	data.TrueNegativeTests.BlockedRequestsNumber = s.TrueNegativeTests.ReqStats.BlockedRequestsNumber
	data.TrueNegativeTests.BypassedRequestsNumber = s.TrueNegativeTests.ReqStats.BypassedRequestsNumber
	data.TrueNegativeTests.UnresolvedRequestsNumber = s.TrueNegativeTests.ReqStats.UnresolvedRequestsNumber
	data.TrueNegativeTests.FailedRequestsNumber = s.TrueNegativeTests.ReqStats.FailedRequestsNumber

	data.TotalSent = data.TruePositiveTests.TotalSent + data.TrueNegativeTests.TotalSent
	data.BlockedRequestsNumber = data.TruePositiveTests.BlockedRequestsNumber + data.TrueNegativeTests.BlockedRequestsNumber
	data.BypassedRequestsNumber = data.TruePositiveTests.BypassedRequestsNumber + data.TrueNegativeTests.BypassedRequestsNumber
	data.UnresolvedRequestsNumber = data.TruePositiveTests.UnresolvedRequestsNumber + data.TrueNegativeTests.UnresolvedRequestsNumber
	data.FailedRequestsNumber = data.TruePositiveTests.FailedRequestsNumber + data.TrueNegativeTests.FailedRequestsNumber

	return data, nil
}

// oncePrepareHTMLFullReport prepares ready data to insert into the HTML template
// once at the first call, and then reuses the previously prepared data
func oncePrepareHTMLFullReport(
	s *db.Statistics, reportTime time.Time, wafName string,
	url string, openApiFile string, args []string, ignoreUnresolved bool, includePayloads bool,
) (*report.HtmlReport, error) {
	var err error

	prepareHTMLReportOnce.Do(func() {
		htmlReportData, err = prepareHTMLFullReport(
			s, reportTime, wafName, url, openApiFile,
			args, ignoreUnresolved, includePayloads,
		)
	})

	return htmlReportData, err
}

// exportFullReportToHtml prepares and saves a full report in HTML format on a disk
// to a temporary file.
func exportFullReportToHtml(
	s *db.Statistics, reportTime time.Time, wafName string,
	url string, openApiFile string, args []string, ignoreUnresolved bool, includePayloads bool,
) (fileName string, err error) {
	reportData, err := oncePrepareHTMLFullReport(s, reportTime, wafName, url, openApiFile, args, ignoreUnresolved, includePayloads)
	if err != nil {
		return "", errors.Wrap(err, "couldn't prepare data for HTML report")
	}

	reportHtml, err := report.RenderFullReportToHTML(reportData)
	if err != nil {
		return "", errors.Wrap(err, "couldn't substitute report data into HTML template")
	}

	file, err := os.CreateTemp("", "gotestwaf_report_*.html")
	if err != nil {
		return "", errors.Wrap(err, "couldn't create a temporary file")
	}
	defer file.Close()

	fileName = file.Name()

	file.Write(reportHtml.Bytes())

	err = os.Chmod(fileName, 0644)

	return fileName, err
}

// printFullReportToHtml prepares and saves a full report in HTML format on a disk.
func printFullReportToHtml(
	s *db.Statistics, reportFile string, reportTime time.Time,
	wafName string, url string, openApiFile string, args []string,
	ignoreUnresolved bool, includePayloads bool,
) error {
	tempFileName, err := exportFullReportToHtml(s, reportTime, wafName, url, openApiFile, args, ignoreUnresolved, includePayloads)
	if err != nil {
		return errors.Wrap(err, "couldn't export report to HTML")
	}

	err = helpers.FileMove(tempFileName, reportFile)
	if err != nil {
		return errors.Wrap(err, "couldn't export report to HTML")
	}

	return nil
}
