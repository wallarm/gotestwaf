package report

import (
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"

	"github.com/wallarm/gotestwaf/internal/db"
)

const (
	colMinWidth = 21
)

func RenderConsoleTable(s *db.Statistics, reportTime time.Time, wafName string, ignoreUnresolved bool) {
	baseHeader := []string{"Test set", "Test case", "Percentage, %", "Blocked", "Bypassed"}
	if !ignoreUnresolved {
		baseHeader = append(baseHeader, "Unresolved")
	}
	baseHeader = append(baseHeader, "Sent", "Failed")

	// Negative cases summary table
	table := tablewriter.NewWriter(os.Stdout)
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

	fmt.Println("\nNegative Tests:")
	table.Render()
	fmt.Println("\nPositive Tests:")
	posTable.Render()
}
