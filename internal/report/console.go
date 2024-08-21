package report

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/db"
)

// The minimum length of each column in a console table report.
const colMinWidth = 21

// RenderConsoleReport prints a console report in selected format.
func RenderConsoleReport(
	s *db.Statistics,
	reportTime time.Time,
	wafName string,
	url string,
	args []string,
	ignoreUnresolved bool,
	format string,
) error {
	switch format {
	case consoleReportTextFormat:
		printConsoleReportTable(s, reportTime, wafName, ignoreUnresolved)
	case consoleReportJsonFormat:
		err := printConsoleReportJson(s, reportTime, wafName, url, args)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown report format: %s", format)
	}

	return nil
}

// printConsoleReportTable prepare and prints a console report in tabular format.
func printConsoleReportTable(
	s *db.Statistics,
	reportTime time.Time,
	wafName string,
	ignoreUnresolved bool,
) {
	baseHeader := []string{"Test set", "Test case", "Percentage, %", "Blocked", "Bypassed"}
	if !ignoreUnresolved {
		baseHeader = append(baseHeader, "Unresolved")
	}
	baseHeader = append(baseHeader, "Sent", "Failed")

	var buffer strings.Builder

	fmt.Fprintf(&buffer, "True-Positive Tests:\n")

	// Negative cases summary table
	table := tablewriter.NewWriter(&buffer)
	table.SetHeader(baseHeader)
	for index := range baseHeader {
		table.SetColMinWidth(index, colMinWidth)
	}

	for _, row := range s.TruePositiveTests.SummaryTable {
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
		fmt.Sprintf("True-Positive Score:\n%.2f%%", s.TruePositiveTests.ResolvedBlockedRequestsPercentage),
		fmt.Sprintf("Blocked (Resolved):\n%d/%d (%.2f%%)",
			s.TruePositiveTests.BlockedRequestsNumber,
			s.TruePositiveTests.ResolvedRequestsNumber,
			s.TruePositiveTests.ResolvedBlockedRequestsPercentage,
		),
		fmt.Sprintf("Bypassed (Resolved):\n%d/%d (%.2f%%)",
			s.TruePositiveTests.BypassedRequestsNumber,
			s.TruePositiveTests.ResolvedRequestsNumber,
			s.TruePositiveTests.ResolvedBypassedRequestsPercentage,
		),
	}
	if !ignoreUnresolved {
		footerNegativeTests = append(footerNegativeTests,
			fmt.Sprintf("Unresolved (Sent):\n%d/%d (%.2f%%)",
				s.TruePositiveTests.UnresolvedRequestsNumber,
				s.TruePositiveTests.AllRequestsNumber,
				s.TruePositiveTests.UnresolvedRequestsPercentage,
			),
		)
	}
	footerNegativeTests = append(footerNegativeTests,
		fmt.Sprintf("Total Sent:\n%d", s.TruePositiveTests.AllRequestsNumber),
		fmt.Sprintf("Failed (Total):\n%d/%d (%.2f%%)",
			s.TruePositiveTests.FailedRequestsNumber,
			s.TruePositiveTests.AllRequestsNumber,
			s.TruePositiveTests.FailedRequestsPercentage,
		),
	)

	table.SetFooter(footerNegativeTests)
	table.Render()

	fmt.Fprintf(&buffer, "\nTrue-Negative Tests:\n")

	// Positive cases summary table
	posTable := tablewriter.NewWriter(&buffer)
	posTable.SetHeader(baseHeader)
	for index := range baseHeader {
		posTable.SetColMinWidth(index, colMinWidth)
	}

	for _, row := range s.TrueNegativeTests.SummaryTable {
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
		fmt.Sprintf("True-Negative Score:\n%.2f%%", s.TrueNegativeTests.ResolvedBypassedRequestsPercentage),
		fmt.Sprintf("Blocked (Resolved):\n%d/%d (%.2f%%)",
			s.TrueNegativeTests.BlockedRequestsNumber,
			s.TrueNegativeTests.ResolvedRequestsNumber,
			s.TrueNegativeTests.ResolvedBlockedRequestsPercentage,
		),
		fmt.Sprintf("Bypassed (Resolved):\n%d/%d (%.2f%%)",
			s.TrueNegativeTests.BypassedRequestsNumber,
			s.TrueNegativeTests.ResolvedRequestsNumber,
			s.TrueNegativeTests.ResolvedBypassedRequestsPercentage,
		),
	}
	if !ignoreUnresolved {
		footerPositiveTests = append(footerPositiveTests,
			fmt.Sprintf("Unresolved (Sent):\n%d/%d (%.2f%%)",
				s.TrueNegativeTests.UnresolvedRequestsNumber,
				s.TrueNegativeTests.AllRequestsNumber,
				s.TrueNegativeTests.UnresolvedRequestsPercentage,
			),
		)
	}
	footerPositiveTests = append(footerPositiveTests,
		fmt.Sprintf("Total Sent:\n%d", s.TrueNegativeTests.AllRequestsNumber),
		fmt.Sprintf("Failed (Total):\n%d/%d (%.2f%%)",
			s.TrueNegativeTests.FailedRequestsNumber,
			s.TrueNegativeTests.AllRequestsNumber,
			s.TrueNegativeTests.FailedRequestsPercentage,
		),
	)

	posTable.SetFooter(footerPositiveTests)
	posTable.Render()

	fmt.Fprintf(&buffer, "\nSummary:\n")

	// summary table
	sumTable := tablewriter.NewWriter(&buffer)
	baseHeader = []string{"Type", "True-Positive tests blocked", "True-Negative tests passed", "Average"}
	sumTable.SetHeader(baseHeader)
	for index := range baseHeader {
		sumTable.SetColMinWidth(index, 27)
	}

	row := []string{"API Security"}
	if s.Score.ApiSec.TruePositive != -1.0 {
		row = append(row, fmt.Sprintf("%.2f%%", s.Score.ApiSec.TruePositive))
	} else {
		row = append(row, "n/a")
	}
	if s.Score.ApiSec.TrueNegative != -1.0 {
		row = append(row, fmt.Sprintf("%.2f%%", s.Score.ApiSec.TrueNegative))
	} else {
		row = append(row, "n/a")
	}
	if s.Score.ApiSec.Average != -1.0 {
		row = append(row, fmt.Sprintf("%.2f%%", s.Score.ApiSec.Average))
	} else {
		row = append(row, "n/a")
	}
	sumTable.Append(row)

	row = []string{"Application Security"}
	if s.Score.AppSec.TruePositive != -1.0 {
		row = append(row, fmt.Sprintf("%.2f%%", s.Score.AppSec.TruePositive))
	} else {
		row = append(row, "n/a")
	}
	if s.Score.AppSec.TrueNegative != -1.0 {
		row = append(row, fmt.Sprintf("%.2f%%", s.Score.AppSec.TrueNegative))
	} else {
		row = append(row, "n/a")
	}
	if s.Score.AppSec.Average != -1.0 {
		row = append(row, fmt.Sprintf("%.2f%%", s.Score.AppSec.Average))
	} else {
		row = append(row, "n/a")
	}
	sumTable.Append(row)

	footer := []string{"", "", "Score"}
	if s.Score.Average != -1.0 {
		footer = append(footer, fmt.Sprintf("%.2f%%", s.Score.Average))
	} else {
		footer = append(footer, "n/a")
	}
	sumTable.SetFooter(footer)
	sumTable.Render()

	fmt.Println(buffer.String())
}

// printConsoleReportJson prepares and prints a console report in json format.
func printConsoleReportJson(
	s *db.Statistics,
	reportTime time.Time,
	wafName string,
	url string,
	args []string,
) error {
	report := jsonReport{
		Date:        reportTime.Format(time.ANSIC),
		ProjectName: wafName,
		URL:         url,
		TestCasesFP: s.TestCasesFingerprint,
		Args:        strings.Join(args, " "),
		Score:       s.Score.Average,
	}

	if len(s.TruePositiveTests.SummaryTable) != 0 {
		report.TruePositiveTests = &testsInfo{
			Score:           s.TruePositiveTests.ResolvedBlockedRequestsPercentage,
			TotalSent:       s.TruePositiveTests.AllRequestsNumber,
			ResolvedTests:   s.TruePositiveTests.ResolvedRequestsNumber,
			BlockedTests:    s.TruePositiveTests.BlockedRequestsNumber,
			BypassedTests:   s.TruePositiveTests.BypassedRequestsNumber,
			UnresolvedTests: s.TruePositiveTests.UnresolvedRequestsNumber,
			FailedTests:     s.TruePositiveTests.FailedRequestsNumber,
			TestSets:        make(testSets),
		}
		for _, row := range s.TruePositiveTests.SummaryTable {
			if report.TruePositiveTests.TestSets[row.TestSet] == nil {
				report.TruePositiveTests.TestSets[row.TestSet] = make(testCases)
			}
			report.TruePositiveTests.TestSets[row.TestSet][row.TestCase] = &testCaseInfo{
				Percentage: row.Percentage,
				Sent:       row.Sent,
				Blocked:    row.Blocked,
				Bypassed:   row.Bypassed,
				Unresolved: row.Unresolved,
				Failed:     row.Failed,
			}
		}
	}

	if len(s.TrueNegativeTests.SummaryTable) != 0 {
		report.TrueNegativeTests = &testsInfo{
			Score:           s.TrueNegativeTests.ResolvedBypassedRequestsPercentage,
			TotalSent:       s.TrueNegativeTests.AllRequestsNumber,
			ResolvedTests:   s.TrueNegativeTests.ResolvedRequestsNumber,
			BlockedTests:    s.TrueNegativeTests.BlockedRequestsNumber,
			BypassedTests:   s.TrueNegativeTests.BypassedRequestsNumber,
			UnresolvedTests: s.TrueNegativeTests.UnresolvedRequestsNumber,
			FailedTests:     s.TrueNegativeTests.FailedRequestsNumber,
			TestSets:        make(testSets),
		}
		for _, row := range s.TrueNegativeTests.SummaryTable {
			if report.TrueNegativeTests.TestSets[row.TestSet] == nil {
				report.TrueNegativeTests.TestSets[row.TestSet] = make(testCases)
			}
			report.TrueNegativeTests.TestSets[row.TestSet][row.TestCase] = &testCaseInfo{
				Percentage: row.Percentage,
				Sent:       row.Sent,
				Blocked:    row.Blocked,
				Bypassed:   row.Bypassed,
				Unresolved: row.Unresolved,
				Failed:     row.Failed,
			}
		}
	}

	jsonBytes, err := json.Marshal(report)
	if err != nil {
		return errors.Wrap(err, "couldn't export report to JSON")
	}

	fmt.Println(string(jsonBytes))

	return nil
}
