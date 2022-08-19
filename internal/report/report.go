package report

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/db"
)

const (
	colMinWidth = 21

	maxReportFilenameLength = 249 // 255 (max length) - 5 (".html") - 1 (to be sure)

	consoleReportTextFormat = "text"
	consoleReportJsonFormat = "json"

	ReportJsonFormat = "json"
	ReportHtmlFormat = "html"
	ReportPdfFormat  = "pdf"
	ReportNoneFormat = "none"
)

type jsonReport struct {
	Date        string `json:"date"`
	ProjectName string `json:"project_name"`
	URL         string `json:"url"`
	TestCasesFP string `json:"fp"`
	Args        string `json:"args"`

	// fields for console report in JSON format
	NegativeTests *testsInfo `json:"negative,omitempty"`
	PositiveTests *testsInfo `json:"positive,omitempty"`

	// fields for full report in JSON format
	Score                 float64       `json:"score,omitempty"`
	Summary               *summary      `json:"summary,omitempty"`
	NegativeTestsPayloads *testPayloads `json:"negative_payloads,omitempty"`
	PositiveTestsPayloads *testPayloads `json:"positive_payloads,omitempty"`
}

type testsInfo struct {
	Score           float64  `json:"score"`
	TotalSent       int      `json:"total_sent"`
	ResolvedTests   int      `json:"resolved_tests"`
	BlockedTests    int      `json:"blocked_tests"`
	BypassedTests   int      `json:"bypassed_tests"`
	UnresolvedTests int      `json:"unresolved_tests"`
	FailedTests     int      `json:"failed_tests"`
	TestSets        testSets `json:"test_sets"`
}

type testSets map[string]testCases

type testCases map[string]*testCaseInfo

type testCaseInfo struct {
	Percentage float64 `json:"percentage"`
	Sent       int     `json:"sent"`
	Blocked    int     `json:"blocked"`
	Bypassed   int     `json:"bypassed"`
	Unresolved int     `json:"unresolved"`
	Failed     int     `json:"failed"`
}

type summary struct {
	NegativeTests *testsInfo `json:"negative,omitempty"`
	PositiveTests *testsInfo `json:"positive,omitempty"`
}

type testPayloads struct {
	Blocked    []*payloadDetails `json:"blocked,omitempty"`
	Bypassed   []*payloadDetails `json:"bypassed,omitempty"`
	Unresolved []*payloadDetails `json:"unresolved,omitempty"`
	Failed     []*payloadDetails `json:"failed,omitempty"`
}

type payloadDetails struct {
	Payload     string `json:"payload"`
	TestSet     string `json:"test_set"`
	TestCase    string `json:"test_case"`
	Encoder     string `json:"encoder"`
	Placeholder string `json:"placeholder"`
	Status      int    `json:"status,omitempty"`

	// Used for non-failed payloads
	AdditionalInformation []string `json:"additional_info,omitempty"`

	// Used for failed payloads
	Reason []string `json:"reason,omitempty"`
}

func ExportFullReport(
	ctx context.Context, s *db.Statistics, reportFile string, reportTime time.Time,
	wafName string, url string, openApiFile string, args string, ignoreUnresolved bool, format string,
) (fullName string, err error) {
	_, reportFileName := filepath.Split(reportFile)
	if len(reportFileName) > maxReportFilenameLength {
		return "", errors.New("report filename too long")
	}

	switch format {
	case ReportHtmlFormat:
		fullName = reportFile + ".html"
		err = printFullReportToHtml(s, fullName, reportTime, wafName, url, openApiFile, args, ignoreUnresolved)
		if err != nil {
			return "", err
		}

	case ReportPdfFormat:
		fullName = reportFile + ".pdf"
		err = printFullReportToPdf(ctx, s, fullName, reportTime, wafName, url, openApiFile, args, ignoreUnresolved)
		if err != nil {
			return "", err
		}

	case ReportJsonFormat:
		fullName = reportFile + ".json"
		err = printFullReportToJson(s, fullName, reportTime, wafName, url, args, ignoreUnresolved)
		if err != nil {
			return "", err
		}

	case ReportNoneFormat:
		return "", nil

	default:
		return "", fmt.Errorf("unknown report format: %s", format)
	}

	return fullName, nil
}

func printFullReportToHtml(
	s *db.Statistics, reportFile string, reportTime time.Time,
	wafName string, url string, openApiFile string, args string, ignoreUnresolved bool,
) error {
	tempFileName, err := exportFullReportToHtml(s, reportTime, wafName, url, openApiFile, args, ignoreUnresolved)
	if err != nil {
		return errors.Wrap(err, "couldn't export report to HTML")
	}

	err = os.Rename(tempFileName, reportFile)
	if err != nil {
		return errors.Wrap(err, "couldn't export report to HTML")
	}

	return nil
}

func printFullReportToPdf(
	ctx context.Context, s *db.Statistics, reportFile string, reportTime time.Time,
	wafName string, url string, openApiFile string, args string, ignoreUnresolved bool,
) error {
	tempFileName, err := exportFullReportToHtml(s, reportTime, wafName, url, openApiFile, args, ignoreUnresolved)
	if err != nil {
		return errors.Wrap(err, "couldn't export report to HTML")
	}

	err = renderToPDF(ctx, tempFileName, reportFile)
	if err != nil {
		return errors.Wrap(err, "couldn't render HTML report to PDF")
	}

	return nil
}

func printFullReportToJson(
	s *db.Statistics, reportFile string, reportTime time.Time,
	wafName string, url string, args string, ignoreUnresolved bool,
) error {
	report := jsonReport{
		Date:        reportTime.Format(time.ANSIC),
		ProjectName: wafName,
		URL:         url,
		Score:       s.WafScore,
		TestCasesFP: s.TestCasesFingerprint,
		Args:        args,
	}

	report.Summary = &summary{}

	if len(s.SummaryTable) != 0 {
		report.Summary.NegativeTests = &testsInfo{
			Score:           s.WafScore,
			TotalSent:       s.AllRequestsNumber,
			ResolvedTests:   s.ResolvedRequestsNumber,
			BlockedTests:    s.BlockedRequestsNumber,
			BypassedTests:   s.BypassedRequestsNumber,
			UnresolvedTests: s.UnresolvedRequestsNumber,
			FailedTests:     s.FailedRequestsNumber,
			TestSets:        make(testSets),
		}
		for _, row := range s.SummaryTable {
			if report.Summary.NegativeTests.TestSets[row.TestSet] == nil {
				report.Summary.NegativeTests.TestSets[row.TestSet] = make(testCases)
			}
			report.Summary.NegativeTests.TestSets[row.TestSet][row.TestCase] = &testCaseInfo{
				Percentage: row.Percentage,
				Sent:       row.Sent,
				Blocked:    row.Blocked,
				Bypassed:   row.Bypassed,
				Unresolved: row.Unresolved,
				Failed:     row.Failed,
			}
		}
	}

	if len(s.PositiveTests.SummaryTable) != 0 {
		report.Summary.PositiveTests = &testsInfo{
			Score:           s.PositiveTests.ResolvedTrueRequestsPercentage,
			TotalSent:       s.PositiveTests.AllRequestsNumber,
			ResolvedTests:   s.PositiveTests.ResolvedRequestsNumber,
			BlockedTests:    s.PositiveTests.BlockedRequestsNumber,
			BypassedTests:   s.PositiveTests.BypassedRequestsNumber,
			UnresolvedTests: s.PositiveTests.UnresolvedRequestsNumber,
			FailedTests:     s.PositiveTests.FailedRequestsNumber,
			TestSets:        make(testSets),
		}
		for _, row := range s.PositiveTests.SummaryTable {
			if report.Summary.PositiveTests.TestSets[row.TestSet] == nil {
				report.Summary.PositiveTests.TestSets[row.TestSet] = make(testCases)
			}
			report.Summary.PositiveTests.TestSets[row.TestSet][row.TestCase] = &testCaseInfo{
				Percentage: row.Percentage,
				Sent:       row.Sent,
				Blocked:    row.Blocked,
				Bypassed:   row.Bypassed,
				Unresolved: row.Unresolved,
				Failed:     row.Failed,
			}
		}
	}

	report.NegativeTestsPayloads = &testPayloads{}

	for _, bypass := range s.Bypasses {
		bypassDetail := &payloadDetails{
			Payload:               bypass.Payload,
			TestSet:               bypass.TestSet,
			TestCase:              bypass.TestCase,
			Encoder:               bypass.Encoder,
			Placeholder:           bypass.Encoder,
			Status:                bypass.ResponseStatusCode,
			AdditionalInformation: bypass.AdditionalInfo,
		}

		report.NegativeTestsPayloads.Bypassed = append(report.NegativeTestsPayloads.Bypassed, bypassDetail)
	}
	if !ignoreUnresolved {
		for _, unresolved := range s.Unresolved {
			unresolvedDetail := &payloadDetails{
				Payload:               unresolved.Payload,
				TestSet:               unresolved.TestSet,
				TestCase:              unresolved.TestCase,
				Encoder:               unresolved.Encoder,
				Placeholder:           unresolved.Encoder,
				Status:                unresolved.ResponseStatusCode,
				AdditionalInformation: unresolved.AdditionalInfo,
			}

			report.NegativeTestsPayloads.Unresolved = append(report.NegativeTestsPayloads.Unresolved, unresolvedDetail)
		}
	}
	for _, failed := range s.Failed {
		failedDetail := &payloadDetails{
			Payload:     failed.Payload,
			TestSet:     failed.TestSet,
			TestCase:    failed.TestCase,
			Encoder:     failed.Encoder,
			Placeholder: failed.Encoder,
			Reason:      failed.Reason,
		}

		report.NegativeTestsPayloads.Failed = append(report.NegativeTestsPayloads.Failed, failedDetail)
	}

	report.PositiveTestsPayloads = &testPayloads{}

	for _, blocked := range s.PositiveTests.FalsePositive {
		blockedDetails := &payloadDetails{
			Payload:               blocked.Payload,
			TestSet:               blocked.TestSet,
			TestCase:              blocked.TestCase,
			Encoder:               blocked.Encoder,
			Placeholder:           blocked.Encoder,
			Status:                blocked.ResponseStatusCode,
			AdditionalInformation: blocked.AdditionalInfo,
		}

		report.PositiveTestsPayloads.Blocked = append(report.PositiveTestsPayloads.Blocked, blockedDetails)
	}
	if !ignoreUnresolved {
		for _, unresolved := range s.PositiveTests.Unresolved {
			unresolvedDetail := &payloadDetails{
				Payload:               unresolved.Payload,
				TestSet:               unresolved.TestSet,
				TestCase:              unresolved.TestCase,
				Encoder:               unresolved.Encoder,
				Placeholder:           unresolved.Encoder,
				Status:                unresolved.ResponseStatusCode,
				AdditionalInformation: unresolved.AdditionalInfo,
			}

			report.PositiveTestsPayloads.Unresolved = append(report.PositiveTestsPayloads.Unresolved, unresolvedDetail)
		}
	}
	for _, failed := range s.PositiveTests.Failed {
		failedDetail := &payloadDetails{
			Payload:     failed.Payload,
			TestSet:     failed.TestSet,
			TestCase:    failed.TestCase,
			Encoder:     failed.Encoder,
			Placeholder: failed.Encoder,
			Reason:      failed.Reason,
		}

		report.PositiveTestsPayloads.Failed = append(report.PositiveTestsPayloads.Failed, failedDetail)
	}

	jsonBytes, err := json.MarshalIndent(report, "", "    ")
	if err != nil {
		return errors.Wrap(err, "couldn't dump report to JSON")
	}

	file, err := os.Create(reportFile)
	if err != nil {
		return errors.Wrap(err, "couldn't create file")
	}
	defer file.Close()

	_, err = file.Write(jsonBytes)
	if err != nil {
		return errors.Wrap(err, "couldn't write report to file")
	}

	return nil
}

func RenderConsoleReport(s *db.Statistics, reportTime time.Time, wafName string, url string, ignoreUnresolved bool, format string) error {
	switch format {
	case consoleReportTextFormat:
		printConsoleReportTable(s, reportTime, wafName, ignoreUnresolved)
	case consoleReportJsonFormat:
		err := printConsoleReportJson(s, reportTime, wafName, url)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown report format: %s", format)
	}

	return nil
}

func printConsoleReportTable(s *db.Statistics, reportTime time.Time, wafName string, ignoreUnresolved bool) {
	baseHeader := []string{"Test set", "Test case", "Percentage, %", "Blocked", "Bypassed"}
	if !ignoreUnresolved {
		baseHeader = append(baseHeader, "Unresolved")
	}
	baseHeader = append(baseHeader, "Sent", "Failed")

	var buffer strings.Builder

	fmt.Fprintf(&buffer, "Negative Tests:\n")

	// Negative cases summary table
	table := tablewriter.NewWriter(&buffer)
	table.SetHeader(baseHeader)
	for index := range baseHeader {
		table.SetColMinWidth(index, colMinWidth)
	}

	for _, row := range s.SummaryTable {
		rowAppend := []string{
			row.TestSet,
			row.TestCase,
			fmt.Sprintf("%.2f", row.Percentage),
			fmt.Sprintf("%d", row.Blocked),
			fmt.Sprintf("%d", row.Bypassed),
		}
		if !ignoreUnresolved {
			rowAppend = append(rowAppend, fmt.Sprintf("%d", row.Unresolved))
		}
		rowAppend = append(rowAppend,
			fmt.Sprintf("%d", row.Sent),
			fmt.Sprintf("%d", row.Failed),
		)
		table.Append(rowAppend)
	}

	footerNegativeTests := []string{
		fmt.Sprintf("Date:\n%s", reportTime.Format("2006-01-02")),
		fmt.Sprintf("Project Name:\n%s", wafName),
		fmt.Sprintf("Average Score:\n%.2f%%", s.WafScore),
		fmt.Sprintf("Blocked (Resolved):\n%d/%d (%.2f%%)",
			s.BlockedRequestsNumber,
			s.ResolvedRequestsNumber,
			s.ResolvedBlockedRequestsPercentage,
		),
		fmt.Sprintf("Bypassed (Resolved):\n%d/%d (%.2f%%)",
			s.BypassedRequestsNumber,
			s.ResolvedRequestsNumber,
			s.ResolvedBypassedRequestsPercentage,
		),
	}
	if !ignoreUnresolved {
		footerNegativeTests = append(footerNegativeTests,
			fmt.Sprintf("Unresolved (Sent):\n%d/%d (%.2f%%)",
				s.UnresolvedRequestsNumber,
				s.AllRequestsNumber,
				s.UnresolvedRequestsPercentage,
			),
		)
	}
	footerNegativeTests = append(footerNegativeTests,
		fmt.Sprintf("Total Sent:\n%d", s.AllRequestsNumber),
		fmt.Sprintf("Failed (Total):\n%d/%d (%.2f%%)",
			s.FailedRequestsNumber,
			s.AllRequestsNumber,
			s.FailedRequestsPercentage,
		),
	)

	table.SetFooter(footerNegativeTests)
	table.Render()

	fmt.Fprintf(&buffer, "\nPositive Tests:\n")

	// Positive cases summary table
	posTable := tablewriter.NewWriter(&buffer)
	posTable.SetHeader(baseHeader)
	for index := range baseHeader {
		posTable.SetColMinWidth(index, colMinWidth)
	}

	for _, row := range s.PositiveTests.SummaryTable {
		rowAppend := []string{
			row.TestSet,
			row.TestCase,
			fmt.Sprintf("%.2f", row.Percentage),
			fmt.Sprintf("%d", row.Blocked),
			fmt.Sprintf("%d", row.Bypassed),
		}
		if !ignoreUnresolved {
			rowAppend = append(rowAppend, fmt.Sprintf("%d", row.Unresolved))
		}
		rowAppend = append(rowAppend,
			fmt.Sprintf("%d", row.Sent),
			fmt.Sprintf("%d", row.Failed),
		)
		posTable.Append(rowAppend)
	}

	footerPositiveTests := []string{
		fmt.Sprintf("Date:\n%s", reportTime.Format("2006-01-02")),
		fmt.Sprintf("Project Name:\n%s", wafName),
		fmt.Sprintf("False Positive Score:\n%.2f%%", s.PositiveTests.ResolvedTrueRequestsPercentage),
		fmt.Sprintf("Blocked (Resolved):\n%d/%d (%.2f%%)",
			s.PositiveTests.BlockedRequestsNumber,
			s.PositiveTests.ResolvedRequestsNumber,
			s.PositiveTests.ResolvedFalseRequestsPercentage,
		),
		fmt.Sprintf("Bypassed (Resolved):\n%d/%d (%.2f%%)",
			s.PositiveTests.BypassedRequestsNumber,
			s.PositiveTests.ResolvedRequestsNumber,
			s.PositiveTests.ResolvedTrueRequestsPercentage,
		),
	}
	if !ignoreUnresolved {
		footerPositiveTests = append(footerPositiveTests,
			fmt.Sprintf("Unresolved (Sent):\n%d/%d (%.2f%%)",
				s.PositiveTests.UnresolvedRequestsNumber,
				s.PositiveTests.AllRequestsNumber,
				s.PositiveTests.UnresolvedRequestsPercentage,
			),
		)
	}
	footerPositiveTests = append(footerPositiveTests,
		fmt.Sprintf("Total Sent:\n%d", s.PositiveTests.AllRequestsNumber),
		fmt.Sprintf("Failed (Total):\n%d/%d (%.2f%%)",
			s.PositiveTests.FailedRequestsNumber,
			s.PositiveTests.AllRequestsNumber,
			s.PositiveTests.FailedRequestsPercentage,
		),
	)

	posTable.SetFooter(footerPositiveTests)
	posTable.Render()

	fmt.Println(buffer.String())
}

func printConsoleReportJson(s *db.Statistics, reportTime time.Time, wafName string, url string) error {
	report := jsonReport{
		Date:        reportTime.Format(time.ANSIC),
		ProjectName: wafName,
		URL:         url,
	}

	if len(s.SummaryTable) != 0 {
		report.NegativeTests = &testsInfo{
			Score:           s.WafScore,
			TotalSent:       s.AllRequestsNumber,
			ResolvedTests:   s.ResolvedRequestsNumber,
			BlockedTests:    s.BlockedRequestsNumber,
			BypassedTests:   s.BypassedRequestsNumber,
			UnresolvedTests: s.UnresolvedRequestsNumber,
			FailedTests:     s.FailedRequestsNumber,
			TestSets:        make(testSets),
		}
		for _, row := range s.SummaryTable {
			if report.NegativeTests.TestSets[row.TestSet] == nil {
				report.NegativeTests.TestSets[row.TestSet] = make(testCases)
			}
			report.NegativeTests.TestSets[row.TestSet][row.TestCase] = &testCaseInfo{
				Percentage: row.Percentage,
				Sent:       row.Sent,
				Blocked:    row.Blocked,
				Bypassed:   row.Bypassed,
				Unresolved: row.Unresolved,
				Failed:     row.Failed,
			}
		}
	}

	if len(s.PositiveTests.SummaryTable) != 0 {
		report.PositiveTests = &testsInfo{
			Score:           s.PositiveTests.ResolvedTrueRequestsPercentage,
			TotalSent:       s.PositiveTests.AllRequestsNumber,
			ResolvedTests:   s.PositiveTests.ResolvedRequestsNumber,
			BlockedTests:    s.PositiveTests.BlockedRequestsNumber,
			BypassedTests:   s.PositiveTests.BypassedRequestsNumber,
			UnresolvedTests: s.PositiveTests.UnresolvedRequestsNumber,
			FailedTests:     s.PositiveTests.FailedRequestsNumber,
			TestSets:        make(testSets),
		}
		for _, row := range s.PositiveTests.SummaryTable {
			if report.PositiveTests.TestSets[row.TestSet] == nil {
				report.PositiveTests.TestSets[row.TestSet] = make(testCases)
			}
			report.PositiveTests.TestSets[row.TestSet][row.TestCase] = &testCaseInfo{
				Percentage: row.Percentage,
				Sent:       row.Sent,
				Blocked:    row.Blocked,
				Bypassed:   row.Bypassed,
				Unresolved: row.Unresolved,
				Failed:     row.Failed,
			}
		}
	}

	jsonBytes, err := json.MarshalIndent(report, "", "    ")
	if err != nil {
		return errors.Wrap(err, "couldn't export report to JSON")
	}

	fmt.Println(string(jsonBytes))

	return nil
}
