package report

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/db"
	"github.com/wallarm/gotestwaf/internal/version"
	"github.com/wallarm/gotestwaf/resources"
)

const (
	MARGECELL = 2 // marge top/bottom of cell

	wallarmLink = "https://wallarm.com/?utm_campaign=gtw_tool&utm_medium=pdf&utm_source=github"

	cellWidth     = 10
	cellHeight    = 10
	lineBreakSize = 10
	pageWidth     = 210
)

func tableClip(pdf *gofpdf.Fpdf, cols []float64, rows [][]string, fontSize float64) {
	pagew, pageh := pdf.GetPageSize()
	_ = pagew
	mleft, mright, mtop, mbottom := pdf.GetMargins()
	_ = mleft
	_ = mright
	_ = mtop

	maxContentWidth := pagew - mleft - mright

	for j, row := range rows {
		_, lineHt := pdf.GetFontSize()
		height := lineHt + MARGECELL

		x, y := pdf.GetXY()

		// Founds max number of lines in the cell to create one size cells in the row.
		nLines := make([]int, len(row))
		var maxNLine int
		for i, txt := range row {
			width := cols[i] * maxContentWidth
			nLines[i] = len(pdf.SplitLines([]byte(txt), width))
			if maxNLine < nLines[i] {
				maxNLine = nLines[i]
			}
		}
		// add a new page if the height of the row doesn't fit on the page
		if y+height*float64(maxNLine) >= pageh-mbottom {
			pdf.AddPage()
			x, y = pdf.GetXY()
		}
		for i, txt := range row {
			if j == 0 {
				pdf.SetFont("Arial", "B", fontSize)
			} else {
				pdf.SetFont("Arial", "", fontSize)
			}
			width := cols[i] * maxContentWidth

			if nLines[i] < maxNLine {
				// draw one line cell with height of highest cell in the row
				pdf.MultiCell(width, height*float64(maxNLine), txt, "1", "", false)
			} else {
				// draw multiline cell with exposed height of one line
				pdf.MultiCell(width, height, txt, "1", "", false)
			}

			x += width
			pdf.SetXY(x, y)
		}
		pdf.Ln(height * float64(maxNLine))
	}
}

func tableClipFailed(pdf *gofpdf.Fpdf, cols []float64, rows [][]string, fontSize float64) {
	pagew, pageh := pdf.GetPageSize()
	_ = pagew
	mleft, mright, mtop, mbottom := pdf.GetMargins()
	_ = mleft
	_ = mright
	_ = mtop

	maxContentWidth := pagew - mleft - mright

	for j := 0; j < len(rows); j += 2 {
		// process row with multiple cells: "Payload", "Test Case", "Encoder", "Placeholder"
		row := rows[j]
		_, lineHt := pdf.GetFontSize()
		height := lineHt + MARGECELL

		x, y := pdf.GetXY()

		// Founds max number of lines in the cell to create one size cells in the row.
		nLines := make([]int, len(row))
		var maxNLine int
		for i, txt := range row {
			width := cols[i] * maxContentWidth
			nLines[i] = len(pdf.SplitLines([]byte(txt), width))
			if maxNLine < nLines[i] {
				maxNLine = nLines[i]
			}
		}
		// add a new page if the height of the row doesn't fit on the page
		if y+height*float64(maxNLine) >= pageh-mbottom {
			pdf.AddPage()
			x, y = pdf.GetXY()
		}
		for i, txt := range row {
			pdf.SetFont("Arial", "", fontSize)

			width := cols[i] * maxContentWidth

			if nLines[i] < maxNLine {
				// draw one line cell with height of highest cell in the row
				pdf.MultiCell(width, height*float64(maxNLine), txt, "1", "", false)
			} else {
				// draw multiline cell with exposed height of one line
				pdf.MultiCell(width, height, txt, "1", "", false)
			}

			x += width
			pdf.SetXY(x, y)
		}
		pdf.Ln(height * float64(maxNLine))

		// process row with single cell: "Reason"
		row = rows[j+1]

		maxNLine = len(pdf.SplitLines([]byte(row[0]), maxContentWidth))

		// add a new page if the height of the row doesn't fit on the page
		if y+height*float64(maxNLine) >= pageh-mbottom {
			pdf.AddPage()
			x, y = pdf.GetXY()
		}

		pdf.MultiCell(maxContentWidth, height, row[0], "1", "", false)
	}
}

func ExportToPDF(s *db.Statistics, reportFile string, reportTime time.Time, wafName string, url string, ignoreUnresolved bool) error {
	summaryHeader := []string{"Test set", "Test case", "Percentage, %", "Blocked", "Bypassed"}
	if !ignoreUnresolved {
		summaryHeader = append(summaryHeader, "Unresolved")
	}
	summaryHeader = append(summaryHeader, "Sent", "Failed")

	baseHeader := []string{"Payload", "Test Case", "Encoder", "Placeholder", "Status"}
	failedHeader := []string{"Payload", "Test Case", "Encoder", "Placeholder"}

	negativeBypassRows := [][]string{baseHeader}
	positiveFalseRows := [][]string{baseHeader}
	positiveTrueRows := [][]string{baseHeader}
	failedRows := [][]string{failedHeader}

	for _, row := range s.Bypasses {
		rowAppend := []string{
			row.Payload,
			row.TestCase,
			row.Encoder,
			row.Placeholder,
			fmt.Sprintf("%d", row.Status),
		}
		negativeBypassRows = append(negativeBypassRows, rowAppend)
	}

	for _, row := range s.PositiveTests.FalsePositive {
		rowAppend := []string{
			row.Payload,
			row.TestCase,
			row.Encoder,
			row.Placeholder,
			fmt.Sprintf("%d", row.Status),
		}
		positiveFalseRows = append(positiveFalseRows, rowAppend)
	}

	for _, row := range s.PositiveTests.TruePositive {
		rowAppend := []string{
			row.Payload,
			row.TestCase,
			row.Encoder,
			row.Placeholder,
			fmt.Sprintf("%d", row.Status),
		}
		positiveTrueRows = append(positiveTrueRows, rowAppend)
	}

	allFailedTests := append(s.Failed[:len(s.Failed):len(s.Failed)], s.PositiveTests.Failed...)
	for _, row := range allFailedTests {
		rowAppend := []string{
			row.Payload,
			row.TestCase,
			row.Encoder,
			row.Placeholder,
		}
		failedRows = append(failedRows, rowAppend, []string{row.Reason})
	}

	// Title page
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.SetFont("Arial", "", 24)
	pdf.Cell(cellWidth, cellHeight, "WAF Testing Results")
	pdf.Ln(lineBreakSize)

	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("WAF Average Score: %.2f%%", s.WafScore))
	pdf.Ln(lineBreakSize / 2)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("WAF Detection Score: %.2f%%", s.ResolvedBlockedRequestsPercentage))
	pdf.Ln(lineBreakSize / 2)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("WAF Positive Tests Score: %.2f%%", s.PositiveTests.ResolvedTrueRequestsPercentage))
	pdf.Ln(lineBreakSize)

	pdf.SetFont("Arial", "", 12)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("WAF Name: %s", wafName))
	pdf.Ln(lineBreakSize / 2)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("WAF URL: %s", url))
	pdf.Ln(lineBreakSize / 2)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("WAF Testing Date: %s", reportTime.Format("02 January 2006")))
	pdf.Ln(lineBreakSize / 2)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("GoTestWAF version:\n%s", version.Version))
	pdf.Ln(lineBreakSize)

	currentY := pdf.GetY()

	// Charts
	onlyNegativeChartFlow := false
	// Show only negative chart if positive chart is not available
	if s.PositiveTests.ResolvedRequestsNumber == 0 {
		onlyNegativeChartFlow = true
	}

	// Negative tests chart
	if s.ResolvedRequestsNumber != 0 {
		chartBuf, err := drawDetectionScoreChart(
			s.BypassedRequestsNumber, s.BlockedRequestsNumber, s.FailedRequestsNumber,
			s.BypassedRequestsNumber+s.BlockedRequestsNumber+s.FailedRequestsNumber,
		)
		if err != nil {
			return errors.Wrap(err, "Plot generation error (negative tests)")
		}
		imageInfo := pdf.RegisterImageReader("Overall Plot", "PNG", chartBuf)
		if pdf.Ok() {
			imgWd, imgHt := imageInfo.Extent()
			imgWd, imgHt = imgWd/2, imgHt/2
			pdf.Image("Overall Plot", pageWidth/20, currentY,
				imgWd, imgHt, onlyNegativeChartFlow, "PNG", 0, "")
		}
	}

	// Positive tests chart
	if s.PositiveTests.ResolvedRequestsNumber != 0 {
		chartFalseBuf, err := drawPositiveTestScoreChart(
			s.PositiveTests.BypassedRequestsNumber, s.PositiveTests.BlockedRequestsNumber, s.PositiveTests.FailedRequestsNumber,
			s.PositiveTests.BypassedRequestsNumber+s.PositiveTests.BlockedRequestsNumber+s.PositiveTests.FailedRequestsNumber,
		)
		if err != nil {
			return errors.Wrap(err, "Plot generation error (positive tests)")
		}
		imageInfoFalse := pdf.RegisterImageReader("False Pos Plot", "PNG", chartFalseBuf)
		if pdf.Ok() {
			imgWd, imgHt := imageInfoFalse.Extent()
			imgWd, imgHt = imgWd/2, imgHt/2
			pdf.Image("False Pos Plot", pageWidth-imgWd-pageWidth/20, currentY,
				imgWd, imgHt, true, "PNG", 0, "")
		}
	}

	// Brief numbers
	unresolvedRequestsNumber := s.UnresolvedRequestsNumber + s.PositiveTests.UnresolvedRequestsNumber
	failedRequestsNumber := s.FailedRequestsNumber + s.PositiveTests.FailedRequestsNumber
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("Total: %v bypasses in %v tests, %v unresolved cases, %v failed cases / %v test cases",
		s.BypassedRequestsNumber, s.ResolvedRequestsNumber, unresolvedRequestsNumber, failedRequestsNumber, s.OverallRequests))
	pdf.Ln(lineBreakSize)

	// Summary table
	summaryTable := [][]string{summaryHeader}
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
		summaryTable = append(summaryTable, rowAppend)
	}

	columns := []float64{0.17, 0.16, 0.16, 0.1, 0.11, 0.13, 0.08, 0.08}
	if ignoreUnresolved {
		columns = append(columns[:5], columns[6:]...)
	}
	tableClip(pdf, columns, summaryTable, 10)

	// Wallarm logo
	reader := bytes.NewReader(resources.WallarmLogo)
	pdf.RegisterImageReader("wallarm-logo", "PNG", reader)
	pdf.Image("wallarm-logo", 15, 280, 20, 0, false, "", 0, wallarmLink)

	// Positive tests page
	pdf.AddPage()
	pdf.SetFont("Arial", "", 24)
	pdf.Cell(cellWidth, cellHeight, "Positive Tests in Details")
	pdf.Ln(lineBreakSize)

	columns = []float64{0.51, 0.15, 0.12, 0.14, 0.08}

	// False Positive payloads block
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(cellWidth, cellHeight,
		fmt.Sprintf("\n%d false positive requests identified as blocked (failed, bad behavior)",
			s.PositiveTests.BlockedRequestsNumber),
	)
	pdf.Ln(lineBreakSize)
	pdf.SetFont("Arial", "", 10)

	tableClip(pdf, columns, positiveFalseRows, 10)

	// True Positive payloads block
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(cellWidth, cellHeight,
		fmt.Sprintf("\n%d true positive requests identified as bypassed (passed, good behavior)",
			s.PositiveTests.BypassedRequestsNumber),
	)
	pdf.Ln(lineBreakSize)
	pdf.SetFont("Arial", "", 10)

	tableClip(pdf, columns, positiveTrueRows, 10)

	// Malicious payloads page
	pdf.AddPage()
	pdf.SetFont("Arial", "", 24)
	pdf.Cell(cellWidth, cellHeight, "Bypasses in Details")
	pdf.Ln(lineBreakSize)

	pdf.SetFont("Arial", "", 12)
	pdf.Cell(cellWidth, cellHeight,
		fmt.Sprintf("\n%d malicious requests have bypassed the WAF", s.BypassedRequestsNumber))
	pdf.Ln(lineBreakSize)

	pdf.SetFont("Arial", "", 10)
	tableClip(pdf, columns, negativeBypassRows, 10)

	if !ignoreUnresolved {
		unresolvedRows := [][]string{baseHeader}
		allUnresolvedTests := append(
			s.Unresolved[:len(s.Unresolved):len(s.Unresolved)],
			s.PositiveTests.Unresolved...)
		for _, row := range allUnresolvedTests {
			rowAppend := []string{
				row.Payload,
				row.TestCase,
				row.Encoder,
				row.Placeholder,
				fmt.Sprintf("%d", row.Status),
			}
			unresolvedRows = append(unresolvedRows, rowAppend)
		}

		pdf.AddPage()
		pdf.SetFont("Arial", "", 24)
		pdf.Cell(cellWidth, cellHeight, "Unresolved Test Cases")
		pdf.Ln(lineBreakSize)
		pdf.SetFont("Arial", "", 12)
		pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("\n%d requests indentified as blocked and passed or as not-blocked and not-passed",
			len(allUnresolvedTests)))
		pdf.Ln(lineBreakSize)
		pdf.SetFont("Arial", "", 10)

		tableClip(pdf, columns, unresolvedRows, 10)
	}

	// Failed requests
	pdf.AddPage()
	pdf.SetFont("Arial", "", 24)
	pdf.Cell(cellWidth, cellHeight, "Failed Test Cases")
	pdf.Ln(lineBreakSize)
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("\n%d failed requests",
		len(allFailedTests)))
	pdf.Ln(lineBreakSize)
	pdf.SetFont("Arial", "", 10)

	columns = []float64{0.59, 0.15, 0.12, 0.14}
	tableClip(pdf, columns, failedRows[:1], 8)
	tableClipFailed(pdf, columns, failedRows[1:], 8)

	if err := pdf.OutputFileAndClose(reportFile); err != nil {
		return errors.Wrap(err, "PDF generation error")
	}

	return nil
}
