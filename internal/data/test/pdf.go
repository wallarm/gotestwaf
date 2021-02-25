package test

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/wcharczuk/go-chart/drawing"

	"github.com/jung-kurt/gofpdf"
	"github.com/jung-kurt/gofpdf/contrib/httpimg"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/wcharczuk/go-chart"
)

const (
	MARGECELL = 2 // marge top/bottom of cell

	wallarmLink = "https://wallarm.com/?utm_campaign=gtw_tool&utm_medium=pdf&utm_source=github"
	trollLink   = "http://troll.wallarm.tools/assets/wallarm.logo.png"

	cellWidth     = 10
	cellHeight    = 10
	lineBreakSize = 10
	pageWidth     = 210
	pageHeight    = 297
)

func tableClip(pdf *gofpdf.Fpdf, cols []float64, rows [][]string, fontSize float64) {
	pagew, pageh := pdf.GetPageSize()
	_ = pagew
	mleft, mright, mtop, mbottom := pdf.GetMargins()
	_ = mleft
	_ = mright
	_ = mtop

	for j, row := range rows {
		_, lineHt := pdf.GetFontSize()
		height := lineHt + MARGECELL

		x, y := pdf.GetXY()

		// Founds max number of lines in the cell to create one size cells in the row.
		nLines := make([]int, len(row))
		var maxNLine int
		for i, txt := range row {
			width := cols[i]
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
			width := cols[i]

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

func drawChart(bypassed int, blocked int, overall int, failed string, passed string) (*bytes.Buffer, error) {
	bypassedProc := float64(bypassed*100) / float64(overall)
	blockedProc := 100.0 - bypassedProc
	pie := chart.PieChart{
		Width:  512,
		Height: 512,
		Values: []chart.Value{
			{
				Value: float64(bypassed),
				Label: fmt.Sprintf("%s - %d (%.2f%%)", failed, bypassed, bypassedProc),
				Style: chart.Style{
					FillColor: drawing.ColorFromAlphaMixedRGBA(234, 67, 54, 255),
				},
			},
			{
				Value: float64(blocked),
				Label: fmt.Sprintf("%s - %d (%.2f%%)", passed, blocked, blockedProc),
				Style: chart.Style{
					FillColor: drawing.ColorFromAlphaMixedRGBA(66, 133, 244, 255),
				},
			},
		},
	}
	buffer := bytes.NewBuffer([]byte{})
	err := pie.Render(chart.PNG, buffer)
	if err != nil {
		return buffer, err
	}
	return buffer, nil
}

func (db *DB) ExportToPDFAndShowTable(reportFile string, reportTime time.Time, WAFName string) error {
	var rows [][]string
	var overallPassedRate, overallTestcasesCompleted float32
	var overallTestsCompleted, overallTestsFailed int

	reportTableTime := reportTime.Format("2006-01-02")
	reportPdfTime := reportTime.Format("02 January 2006")

	rows = append(rows, []string{"Test set", "Test case", "Percentage, %", "Passed/Blocked", "Failed/Bypassed"})

	sortedTestSets := make([]string, 0, len(db.counters))
	for testSet := range db.counters {
		sortedTestSets = append(sortedTestSets, testSet)
	}
	sort.Strings(sortedTestSets)

	for _, testSet := range sortedTestSets {
		sortedTestCases := make([]string, 0, len(db.counters[testSet]))
		for testCase := range db.counters[testSet] {
			sortedTestCases = append(sortedTestCases, testCase)
		}
		sort.Strings(sortedTestCases)
		for _, testCase := range sortedTestCases {
			passed := db.counters[testSet][testCase][true]
			failed := db.counters[testSet][testCase][false]
			total := passed + failed
			overallTestsCompleted += total
			overallTestsFailed += failed

			var percentage float32 = 0
			if total != 0 {
				percentage = float32(passed) / float32(total) * 100
			}
			// Invert the score for the false positive test sets
			if strings.Contains(testSet, "false") {
				percentage = 100 - percentage
			}

			rows = append(rows,
				[]string{
					testSet,
					testCase,
					fmt.Sprintf("%.2f", percentage),
					fmt.Sprintf("%d", passed),
					fmt.Sprintf("%d", failed)},
			)
			overallTestcasesCompleted += 1.00
			overallPassedRate += percentage
		}
	}

	wafScore := overallPassedRate / overallTestcasesCompleted

	// Create a table.
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Test Set", "Test Case", "Percentage, %", "Passed/Blocked", "Failed/Bypassed"})
	table.SetFooter([]string{fmt.Sprintf("Date: %s", reportTableTime), "WAF Name:", WAFName, "WAF Score:", fmt.Sprintf("%.2f%%", wafScore)})

	for _, v := range rows[1:] {
		table.Append(v)
	}
	table.Render()

	// Create a pdf file
	cols := []float64{35, 45, 35, 35, 40}
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "", 24)
	pdf.Cell(cellWidth, cellHeight, "WAF Testing Results")

	pdf.Ln(lineBreakSize)
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("WAF score: %.2f%%", wafScore))
	pdf.SetFont("Arial", "", 12)
	pdf.Ln(lineBreakSize / 2)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("WAF name: %s", WAFName))
	pdf.Ln(lineBreakSize / 2)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("WAF testing date: %s", reportPdfTime))
	pdf.Ln(lineBreakSize)

	var rowsPayloads [][]string
	var rowsTruePos [][]string
	var rowsFalsePos [][]string

	rowsPayloads = append(rowsPayloads, []string{"Payload", "Test Case", "Encoder", "Placeholder", "Status"})
	// True positive  - false positive payloads that bypass the WAF (good behavior)
	rowsTruePos = append(rowsTruePos, []string{"Payload", "Test Case", "Encoder", "Placeholder", "Status"})
	// False positive - false positive payloads that were blocked (bad behavior)
	rowsFalsePos = append(rowsFalsePos, []string{"Payload", "Test Case", "Encoder", "Placeholder", "Status"})

	for _, failedTest := range db.failedTests {
		payload := fmt.Sprintf("%+q", failedTest.Payload)
		payload = strings.ReplaceAll(payload[1:len(payload)-1], `\"`, `"`)
		toAppend := []string{payload,
			failedTest.Case,
			failedTest.Encoder,
			failedTest.Placeholder,
			strconv.Itoa(failedTest.ResponseStatusCode)}
		// Failed for false pos - blocked by the waf, bad behavior
		if strings.Contains(failedTest.Set, "false") {
			rowsFalsePos = append(rowsFalsePos, toAppend)
		} else {
			rowsPayloads = append(rowsPayloads, toAppend)
		}
	}

	// Num of bypasses = (failed tests) - (false pos and true pos) - (NA tests (unknown results))
	// Or, in other words, num of bypasses = correct malicious bypasses only
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("%v bypasses in %v tests, %v unresolved cases / %v test cases",
		len(rowsPayloads)-1, overallTestsCompleted, len(db.naTests), overallTestcasesCompleted))
	pdf.Ln(lineBreakSize)

	tableClip(pdf, cols, rows, 12)

	cols = []float64{100, 30, 20, 25, 15}

	for _, blockedTest := range db.passedTests {
		payload := fmt.Sprintf("%+q", blockedTest.Payload)
		payload = strings.ReplaceAll(payload[1:len(payload)-1], `\"`, `"`)
		toAppend := []string{payload,
			blockedTest.Case,
			blockedTest.Encoder,
			blockedTest.Placeholder,
			strconv.Itoa(blockedTest.ResponseStatusCode)}
		// Passed for false pos - bypassed, good behavior
		if strings.Contains(blockedTest.Set, "false") {
			rowsTruePos = append(rowsTruePos, toAppend)
		}
	}

	pdf.Ln(lineBreakSize)

	// Num = number of actual rows - top header (1 line)
	truePosNum := len(rowsTruePos) - 1
	falsePosNum := len(rowsFalsePos) - 1
	// Include only real bypasses, without unknown or false pos/true pos
	bypassesNum := len(rowsPayloads) - 1
	blockedNum := len(db.passedTests) - truePosNum
	currentY := pdf.GetY()

	chartBuf, err := drawChart(bypassesNum, blockedNum, bypassesNum+blockedNum, "Bypassed", "Blocked")
	if err != nil {
		return errors.Wrap(err, "Plot generation error")
	}
	imageInfo := pdf.RegisterImageReader("Overall Plot", "PNG", chartBuf)
	if pdf.Ok() {
		imgWd, imgHt := imageInfo.Extent()
		imgWd, imgHt = imgWd/2, imgHt/2
		pdf.Image("Overall Plot", pageWidth/20, currentY,
			imgWd, imgHt, false, "PNG", 0, "")
	}

	chartFalseBuf, err := drawChart(falsePosNum, truePosNum, truePosNum+falsePosNum, "False Positive", "True Positive")
	if err != nil {
		return errors.Wrap(err, "Plot generation error")
	}
	imageInfoFalse := pdf.RegisterImageReader("False Pos Plot", "PNG", chartFalseBuf)
	if pdf.Ok() {
		imgWd, imgHt := imageInfoFalse.Extent()
		imgWd, imgHt = imgWd/2, imgHt/2
		pdf.Image("False Pos Plot", pageWidth-imgWd-pageWidth/20, currentY,
			imgWd, imgHt, false, "PNG", 0, "")
	}

	httpimg.Register(pdf, trollLink, "")
	pdf.Image(trollLink, 15, 280, 20, 0, false, "", 0, wallarmLink)

	pdf.AddPage()

	// Malicious payloads block
	pdf.SetFont("Arial", "", 24)
	pdf.Cell(cellWidth, cellHeight, "Bypasses in Details")
	pdf.Ln(lineBreakSize)
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("\n%d malicious requests have bypassed the WAF", len(rowsPayloads)-1))
	pdf.Ln(lineBreakSize)
	pdf.SetFont("Arial", "", 10)

	tableClip(pdf, cols, rowsPayloads, 10)

	pdf.AddPage()
	pdf.SetFont("Arial", "", 24)
	pdf.Cell(cellWidth, cellHeight, "False Positive and True Positive in Details")
	pdf.Ln(lineBreakSize)

	// False Positive payloads block
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("\n%d false positive requests identified as blocked (failed, bad behavior)", len(rowsFalsePos)-1))
	pdf.Ln(lineBreakSize)
	pdf.SetFont("Arial", "", 10)

	tableClip(pdf, cols, rowsFalsePos, 10)

	// True Positive payloads block
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("\n%d true positive requests identified as bypassed (passed, good behavior)", len(rowsTruePos)-1))
	pdf.Ln(lineBreakSize)
	pdf.SetFont("Arial", "", 10)

	tableClip(pdf, cols, rowsTruePos, 10)

	pdf.AddPage()

	cols = []float64{100, 30, 20, 25, 15}
	rows = [][]string{}

	rows = append(rows, []string{"Payload", "Test Case", "Encoder", "Placeholder", "Status"})
	for _, naTest := range db.naTests {
		payload := fmt.Sprintf("%+q", naTest.Payload)
		payload = strings.ReplaceAll(payload[1:len(payload)-1], `\"`, `"`)
		rows = append(rows,
			[]string{payload,
				naTest.Case,
				naTest.Encoder,
				naTest.Placeholder,
				strconv.Itoa(naTest.ResponseStatusCode)},
		)
	}
	pdf.SetFont("Arial", "", 24)
	pdf.Cell(cellWidth, cellHeight, "Unresolved Test Cases")
	pdf.Ln(lineBreakSize)
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("\n%d requests indentified as blocked and passed or as not-blocked and not-passed",
		len(db.naTests)))
	pdf.Ln(lineBreakSize)
	pdf.SetFont("Arial", "", 10)

	tableClip(pdf, cols, rows, 10)

	err = pdf.OutputFileAndClose(reportFile)
	if err != nil {
		return errors.Wrap(err, "PDF generation error")
	}

	fmt.Printf("\nPDF report is ready: %s\n", reportFile)
	return nil
}
