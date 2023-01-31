package report

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/db"
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
	}
)

func computeGrade(value float64, all int) *report.Grade {
	g := &report.Grade{
		Percentage:     0.0,
		Mark:           naMark,
		CSSClassSuffix: "na",
	}

	if all == 0 {
		return g
	}

	g.Percentage = value / float64(all)
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
	data.ApiSec.TrueNegative = computeGrade(float64(apiSecNegBlockedNum), apiSecNegNum)
	data.ApiSec.TruePositive = computeGrade(float64(apiSecPosBypassNum), apiSecPosNum)
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

	data.AppSec.TrueNegative = computeGrade(float64(appSecNegBlockedNum), appSecNegNum)
	data.AppSec.TruePositive = computeGrade(float64(appSecPosBypassNum), appSecPosNum)
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

	apiIndicators, apiItems, appIndicators, appItems := generateChartData(s)

	data.ApiSecChartData.Indicators = apiIndicators
	data.ApiSecChartData.Items = apiItems
	data.AppSecChartData.Indicators = appIndicators
	data.AppSecChartData.Items = appItems

	data.NegativeTests.SummaryTable = make(map[string]*report.TestSetSummary)
	for _, row := range s.NegativeTests.SummaryTable {
		if _, ok := data.NegativeTests.SummaryTable[row.TestSet]; !ok {
			data.NegativeTests.SummaryTable[row.TestSet] = &report.TestSetSummary{}
		}

		testSetSum := data.NegativeTests.SummaryTable[row.TestSet]

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
	for _, testSetSum := range data.NegativeTests.SummaryTable {
		testSetSum.Percentage = db.Round(testSetSum.Percentage / float64(testSetSum.ResolvedTestCasesNumber))
	}

	data.PositiveTests.SummaryTable = make(map[string]*report.TestSetSummary)
	for _, row := range s.PositiveTests.SummaryTable {
		if _, ok := data.PositiveTests.SummaryTable[row.TestSet]; !ok {
			data.PositiveTests.SummaryTable[row.TestSet] = &report.TestSetSummary{}
		}

		testSetSum := data.PositiveTests.SummaryTable[row.TestSet]

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
	for _, testSetSum := range data.PositiveTests.SummaryTable {
		testSetSum.Percentage = db.Round(testSetSum.Percentage / float64(testSetSum.ResolvedTestCasesNumber))
	}

	if includePayloads {
		// map[paths]map[payload]map[statusCode]*testDetails
		negBypassed := make(map[string]map[string]map[int]*report.TestDetails)
		for _, d := range s.NegativeTests.Bypasses {
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
		for _, d := range s.NegativeTests.Unresolved {
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

		data.NegativeTests.Bypassed = negBypassed
		data.NegativeTests.Unresolved = negUnresolved
		data.NegativeTests.Failed = s.NegativeTests.Failed

		// map[payload]map[statusCode]*testDetails
		posBlocked := make(map[string]map[int]*report.TestDetails)
		for _, d := range s.PositiveTests.FalsePositive {
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
		for _, d := range s.PositiveTests.TruePositive {
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
		for _, d := range s.PositiveTests.Unresolved {
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

		data.PositiveTests.Blocked = posBlocked
		data.PositiveTests.Bypassed = posBypassed
		data.PositiveTests.Unresolved = posUnresolved
		data.PositiveTests.Failed = s.PositiveTests.Failed
	}

	data.ScannedPaths = s.Paths

	data.NegativeTests.Percentage = s.NegativeTests.ResolvedBlockedRequestsPercentage
	data.NegativeTests.TotalSent = s.NegativeTests.AllRequestsNumber
	data.NegativeTests.BlockedRequestsNumber = s.NegativeTests.BlockedRequestsNumber
	data.NegativeTests.BypassedRequestsNumber = s.NegativeTests.BypassedRequestsNumber
	data.NegativeTests.UnresolvedRequestsNumber = s.NegativeTests.UnresolvedRequestsNumber
	data.NegativeTests.FailedRequestsNumber = s.NegativeTests.FailedRequestsNumber

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

	err = os.Rename(tempFileName, reportFile)
	if err != nil {
		return errors.Wrap(err, "couldn't export report to HTML")
	}

	return nil
}
