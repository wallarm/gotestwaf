package report

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/db"
)

const (
	colMinWidth = 21

	reportTextFormat = "text"
	reportJsonFormat = "json"
)

type JsonReport struct {
	Date          string     `json:"date"`
	ProjectName   string     `json:"project_name"`
	URL           string     `json:"url"`
	NegativeTests *TestsInfo `json:"negative,omitempty"`
	PositiveTests *TestsInfo `json:"positive,omitempty"`
}

type TestsInfo struct {
	Score           string   `json:"score"`
	TotalSent       int      `json:"total_sent"`
	ResolvedTests   int      `json:"resolved_tests"`
	BlockedTests    int      `json:"blocked_tests"`
	BypassedTests   int      `json:"bypassed_tests"`
	UnresolvedTests int      `json:"unresolved_tests"`
	FailedTests     int      `json:"failed_tests"`
	TestSets        TestSets `json:"test_sets"`
}

type TestSets map[string]TestCases

type TestCases map[string]*TestCaseInfo

type TestCaseInfo struct {
	Percentage float32 `json:"percentage"`
	Sent       int     `json:"sent"`
	Blocked    int     `json:"blocked"`
	Bypassed   int     `json:"bypassed"`
	Unresolved int     `json:"unresolved"`
	Failed     int     `json:"failed"`
}

func RenderConsoleReport(s *db.Statistics, reportTime time.Time, wafName string, url string, ignoreUnresolved bool, format string) error {
	switch format {
	case reportTextFormat:
		renderTable(s, reportTime, wafName, ignoreUnresolved)
	case reportJsonFormat:
		err := renderJson(s, reportTime, wafName, url)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown report format: %s", format)
	}

	return nil
}

func renderTable(s *db.Statistics, reportTime time.Time, wafName string, ignoreUnresolved bool) {
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
	posTable := tablewriter.NewWriter(os.Stdout)
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

func renderJson(s *db.Statistics, reportTime time.Time, wafName string, url string) error {
	report := JsonReport{
		Date:        reportTime.Format(time.ANSIC),
		ProjectName: wafName,
		URL:         url,
	}

	if len(s.SummaryTable) != 0 {
		report.NegativeTests = &TestsInfo{
			Score:           fmt.Sprintf("%.2f%%", s.WafScore),
			TotalSent:       s.AllRequestsNumber,
			ResolvedTests:   s.ResolvedRequestsNumber,
			BlockedTests:    s.BlockedRequestsNumber,
			BypassedTests:   s.BypassedRequestsNumber,
			UnresolvedTests: s.UnresolvedRequestsNumber,
			FailedTests:     s.FailedRequestsNumber,
			TestSets:        make(TestSets),
		}
		for _, row := range s.SummaryTable {
			if report.NegativeTests.TestSets[row.TestSet] == nil {
				report.NegativeTests.TestSets[row.TestSet] = make(TestCases)
			}
			report.NegativeTests.TestSets[row.TestSet][row.TestCase] = &TestCaseInfo{
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
		report.PositiveTests = &TestsInfo{
			Score:           fmt.Sprintf("%.2f%%", s.WafScore),
			TotalSent:       s.PositiveTests.AllRequestsNumber,
			ResolvedTests:   s.PositiveTests.ResolvedRequestsNumber,
			BlockedTests:    s.PositiveTests.BlockedRequestsNumber,
			BypassedTests:   s.PositiveTests.BypassedRequestsNumber,
			UnresolvedTests: s.PositiveTests.UnresolvedRequestsNumber,
			FailedTests:     s.PositiveTests.FailedRequestsNumber,
			TestSets:        make(TestSets),
		}
		for _, row := range s.PositiveTests.SummaryTable {
			if report.PositiveTests.TestSets[row.TestSet] == nil {
				report.PositiveTests.TestSets[row.TestSet] = make(TestCases)
			}
			report.PositiveTests.TestSets[row.TestSet][row.TestCase] = &TestCaseInfo{
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
		return errors.Wrap(err, "couldn't dump report to JSON")
	}

	fmt.Println(string(jsonBytes))

	return nil
}
