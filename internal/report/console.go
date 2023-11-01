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

	fmt.Fprintf(&buffer, "Negative Tests:\n")

	// Negative cases summary table
	table := tablewriter.NewWriter(&buffer)
	table.SetHeader(baseHeader)
	for index := range baseHeader {
		table.SetColMinWidth(index, colMinWidth)
	}

	for _, row := range s.NegativeTests.SummaryTable {
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
		fmt.Sprintf("True Negative Score:\n%.2f%%", s.NegativeTests.ResolvedBlockedRequestsPercentage),
		fmt.Sprintf("Blocked (Resolved):\n%d/%d (%.2f%%)",
			s.NegativeTests.BlockedRequestsNumber,
			s.NegativeTests.ResolvedRequestsNumber,
			s.NegativeTests.ResolvedBlockedRequestsPercentage,
		),
		fmt.Sprintf("Bypassed (Resolved):\n%d/%d (%.2f%%)",
			s.NegativeTests.BypassedRequestsNumber,
			s.NegativeTests.ResolvedRequestsNumber,
			s.NegativeTests.ResolvedBypassedRequestsPercentage,
		),
	}
	if !ignoreUnresolved {
		footerNegativeTests = append(footerNegativeTests,
			fmt.Sprintf("Unresolved (Sent):\n%d/%d (%.2f%%)",
				s.NegativeTests.UnresolvedRequestsNumber,
				s.NegativeTests.AllRequestsNumber,
				s.NegativeTests.UnresolvedRequestsPercentage,
			),
		)
	}
	footerNegativeTests = append(footerNegativeTests,
		fmt.Sprintf("Total Sent:\n%d", s.NegativeTests.AllRequestsNumber),
		fmt.Sprintf("Failed (Total):\n%d/%d (%.2f%%)",
			s.NegativeTests.FailedRequestsNumber,
			s.NegativeTests.AllRequestsNumber,
			s.NegativeTests.FailedRequestsPercentage,
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

	fmt.Fprintf(&buffer, "\nSummary:\n")

	// summary table
	sumTable := tablewriter.NewWriter(&buffer)
	baseHeader = []string{"Type", "True-negative tests blocked", "True-positive tests passed", "Average"}
	sumTable.SetHeader(baseHeader)
	for index := range baseHeader {
		sumTable.SetColMinWidth(index, 27)
	}

	row := []string{"API Security"}
	if s.Score.ApiSec.TrueNegative != -1.0 {
		row = append(row, fmt.Sprintf("%.2f%%", s.Score.ApiSec.TrueNegative))
	} else {
		row = append(row, "n/a")
	}
	if s.Score.ApiSec.TruePositive != -1.0 {
		row = append(row, fmt.Sprintf("%.2f%%", s.Score.ApiSec.TruePositive))
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
	if s.Score.AppSec.TrueNegative != -1.0 {
		row = append(row, fmt.Sprintf("%.2f%%", s.Score.AppSec.TrueNegative))
	} else {
		row = append(row, "n/a")
	}
	if s.Score.AppSec.TruePositive != -1.0 {
		row = append(row, fmt.Sprintf("%.2f%%", s.Score.AppSec.TruePositive))
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

	if len(s.NegativeTests.SummaryTable) != 0 {
		report.NegativeTests = &testsInfo{
			Score:           s.NegativeTests.ResolvedBlockedRequestsPercentage,
			TotalSent:       s.NegativeTests.AllRequestsNumber,
			ResolvedTests:   s.NegativeTests.ResolvedRequestsNumber,
			BlockedTests:    s.NegativeTests.BlockedRequestsNumber,
			BypassedTests:   s.NegativeTests.BypassedRequestsNumber,
			UnresolvedTests: s.NegativeTests.UnresolvedRequestsNumber,
			FailedTests:     s.NegativeTests.FailedRequestsNumber,
			TestSets:        make(testSets),
		}
		for _, row := range s.NegativeTests.SummaryTable {
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

	jsonBytes, err := json.Marshal(report)
	if err != nil {
		return errors.Wrap(err, "couldn't export report to JSON")
	}

	fmt.Println(string(jsonBytes))

	return nil
}
