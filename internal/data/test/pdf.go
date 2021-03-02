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
	colMinWidth   = 21
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

func drawChart(bypassed int, blocked int, overall int, failed string, passed string, title string) (*bytes.Buffer, error) {
	bypassedProc := float64(bypassed*100) / float64(overall)
	blockedProc := 100.0 - bypassedProc
	pie := chart.PieChart{
		DPI:   85,
		Title: fmt.Sprintf("%s", title),
		TitleStyle: chart.Style{
			Show:              true,
			TextVerticalAlign: chart.TextVerticalAlignBaseline,
		},
		Background: chart.Style{
			Show:    true,
			Padding: chart.NewBox(25, 25, 25, 25),
		},
		Width:  512,
		Height: 512,
		Values: []chart.Value{
			{
				Value: float64(bypassed),
				Label: fmt.Sprintf("%s: %d (%.2f%%)", failed, bypassed, bypassedProc),
				Style: chart.Style{
					// Red
					FillColor: drawing.ColorFromAlphaMixedRGBA(66, 133, 244, 255),
					FontSize:  12,
				},
			},
			{
				Value: float64(blocked),
				Label: fmt.Sprintf("%s: %d (%.2f%%)", passed, blocked, blockedProc),
				Style: chart.Style{
					// Blue
					FillColor: drawing.ColorFromAlphaMixedRGBA(234, 67, 54, 255),
					FontSize:  12,
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

func calculatePercentage(first int, second int) (percentage float32) {
	percentage = float32(first) / float32(second) * 100
	return
}

func (db *DB) RenderTable(reportTime time.Time, WAFName string) ([][]string, error) {
	baseHeader := []string{"Test set", "Test case", "Percentage, %", "Blocked", "Bypassed", "Unresolved"}

	// Table rows to render, regular and positive cases
	positiveRows := [][]string{baseHeader}
	regularRows := [][]string{baseHeader}

	// Counters to use with table footers
	positiveCasesNum := make(map[bool]int)
	regularCasesNum := make(map[string]int)

	unresolvedCasesNum := make(map[string]int)
	var unresolvedPositiveCasesNum int
	for _, naTest := range db.naTests {
		if strings.Contains(naTest.Set, "false") {
			unresolvedPositiveCasesNum += 1
		} else {
			unresolvedCasesNum[naTest.Case] += 1
		}
	}

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
			unresolved := unresolvedCasesNum[testCase]
			passed := db.counters[testSet][testCase][true]
			// Avoid the unresolved cases when counting failed
			failed := db.counters[testSet][testCase][false] - unresolved
			// But include the unresolved results in total score to calculate them
			total := passed + failed + unresolved
			totalResolved := passed + failed
			db.overallTestsCompleted += total
			db.overallTestsFailed += failed

			var percentage float32 = 0
			if totalResolved != 0 {
				percentage = float32(passed) / float32(totalResolved) * 100
			}

			// If positive set - move to another table (remove from general cases)
			if strings.Contains(testSet, "false") {
				// False positive - blocked by the WAF (bad behavior, failed)
				positiveCasesNum[false] += failed
				// True positive - bypassed (good behavior, passed)
				positiveCasesNum[true] += passed

				// Swap the "failed" and "passed" cases for positive cases
				rowAppend := []string{
					testSet,
					testCase,
					fmt.Sprintf("%.2f", percentage),
					fmt.Sprintf("%d", failed),
					fmt.Sprintf("%d", passed),
					fmt.Sprintf("%d", unresolvedCasesNum[testCase])}

				positiveRows = append(positiveRows, rowAppend)
				continue
			}

			// If not positive set - insert into the original table, update stats
			rowAppend := []string{
				testSet,
				testCase,
				fmt.Sprintf("%.2f", percentage),
				fmt.Sprintf("%d", passed),
				fmt.Sprintf("%d", failed),
				fmt.Sprintf("%d", unresolvedCasesNum[testCase])}

			regularCasesNum["blocked"] += passed
			regularCasesNum["bypassed"] += failed

			regularRows = append(regularRows, rowAppend)

			db.overallTestcasesCompleted += 1.00
			db.overallPassedRate += percentage
		}
	}

	db.wafScore = db.overallPassedRate / db.overallTestcasesCompleted

	// Create a table for regular cases (excluding positive cases)
	fmt.Println("\nNegative Tests:")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(baseHeader)

	for _, row := range regularRows[1:] {
		table.Append(row)
	}
	for index := range baseHeader {
		table.SetColMinWidth(index, colMinWidth)
	}

	positiveTestsSum := positiveCasesNum[false] + positiveCasesNum[true]
	resolvedTestsSum := db.overallTestsCompleted - len(db.naTests) - positiveTestsSum

	unresolvedRate := calculatePercentage(len(db.naTests), db.overallTestsCompleted)
	blockedRate := calculatePercentage(regularCasesNum["blocked"], resolvedTestsSum)
	bypassedRate := calculatePercentage(regularCasesNum["bypassed"], resolvedTestsSum)

	table.SetFooter([]string{
		fmt.Sprintf("Date:\n%s", reportTime.Format("2006-01-02")),
		fmt.Sprintf("WAF Name:\n%s", WAFName),
		fmt.Sprintf("WAF Average Score:\n%.2f%%", db.wafScore),
		fmt.Sprintf("Blocked (Resolved):\n%d/%d (%.2f%%)", regularCasesNum["blocked"], resolvedTestsSum, blockedRate),
		fmt.Sprintf("Bypassed (Resolved):\n%d/%d (%.2f%%)", regularCasesNum["bypassed"], resolvedTestsSum, bypassedRate),
		fmt.Sprintf("Unresolved:\n%d/%d (%.2f%%)", len(db.naTests), db.overallTestsCompleted, unresolvedRate)})
	table.Render()

	// Create a table for positive cases
	fmt.Println("\nPositive Tests:")
	posTable := tablewriter.NewWriter(os.Stdout)
	posTable.SetHeader(baseHeader)

	for _, row := range positiveRows[1:] {
		posTable.Append(row)
	}
	for index := range baseHeader {
		posTable.SetColMinWidth(index, colMinWidth)
	}

	unresolvedPosRate := calculatePercentage(unresolvedPositiveCasesNum, positiveTestsSum)
	resolvedPositiveTests := positiveTestsSum - unresolvedPositiveCasesNum
	falsePosRate := calculatePercentage(positiveCasesNum[false], resolvedPositiveTests)
	truePosRate := calculatePercentage(positiveCasesNum[true], resolvedPositiveTests)

	posTable.SetFooter([]string{
		fmt.Sprintf("Date:\n%s", reportTime.Format("2006-01-02")),
		fmt.Sprintf("WAF Name:\n%s", WAFName),
		fmt.Sprintf("WAF Positive Score:\n%.2f%%", truePosRate),
		fmt.Sprintf("False positive (res):\n%d/%d (%.2f%%)", positiveCasesNum[false], resolvedPositiveTests, falsePosRate),
		fmt.Sprintf("True positive (res):\n%d/%d (%.2f%%)", positiveCasesNum[true], resolvedPositiveTests, truePosRate),
		fmt.Sprintf("Unresolved:\n%d/%d (%.2f%%)", unresolvedPositiveCasesNum, positiveTestsSum, unresolvedPosRate)})
	posTable.Render()

	return regularRows, nil
}

func (db *DB) ExportToPDF(reportFile string, reportTime time.Time, WAFName string, url string, rows [][]string) error {
	baseHeader := []string{"Payload", "Test Case", "Encoder", "Placeholder", "Status"}

	maliciousRows := [][]string{baseHeader}
	truePosRows := [][]string{baseHeader}
	falsePosRows := [][]string{baseHeader}

	for _, failedTest := range db.failedTests {
		payload := fmt.Sprintf("%+q", failedTest.Payload)
		payload = strings.ReplaceAll(payload[1:len(payload)-1], `\"`, `"`)
		toAppend := []string{payload,
			failedTest.Case,
			failedTest.Encoder,
			failedTest.Placeholder,
			strconv.Itoa(failedTest.ResponseStatusCode)}
		// Failed for False Positive - blocked by the waf (bad behavior)
		if strings.Contains(failedTest.Set, "false") {
			falsePosRows = append(falsePosRows, toAppend)
			// Failed for malicious payload - bypass (bad behavior)
		} else {
			maliciousRows = append(maliciousRows, toAppend)
		}
	}

	for _, blockedTest := range db.passedTests {
		payload := fmt.Sprintf("%+q", blockedTest.Payload)
		payload = strings.ReplaceAll(payload[1:len(payload)-1], `\"`, `"`)
		// Passed for false pos - bypassed (good behavior)
		if strings.Contains(blockedTest.Set, "false") {
			truePosRows = append(truePosRows, []string{payload,
				blockedTest.Case,
				blockedTest.Encoder,
				blockedTest.Placeholder,
				strconv.Itoa(blockedTest.ResponseStatusCode)})
		}
	}

	// Num (general): number of actual rows minus top header (1 line)
	truePosNum := len(truePosRows) - 1
	falsePosNum := len(falsePosRows) - 1
	// Include only real bypasses, without unknown or positive cases
	bypassesNum := len(maliciousRows) - 1
	blockedNum := len(db.passedTests) - truePosNum

	cols := []float64{25, 35, 30, 35, 35, 30}

	// Title page
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.SetFont("Arial", "", 24)
	pdf.Cell(cellWidth, cellHeight, "WAF Testing Results")
	pdf.Ln(lineBreakSize)

	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("WAF Average Score: %.2f%%", db.wafScore))
	pdf.SetFont("Arial", "", 12)
	pdf.Ln(lineBreakSize / 2)

	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("WAF Detection Score: %.2f%%", calculatePercentage(blockedNum, bypassesNum+blockedNum)))
	pdf.SetFont("Arial", "", 12)
	pdf.Ln(lineBreakSize / 2)

	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("WAF Positive Tests Score: %.2f%%", calculatePercentage(truePosNum, truePosNum+falsePosNum)))
	pdf.SetFont("Arial", "", 12)
	pdf.Ln(lineBreakSize)

	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("WAF Name: %s", WAFName))
	pdf.Ln(lineBreakSize / 2)

	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("WAF URL: %s", url))
	pdf.Ln(lineBreakSize / 2)

	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("WAF Testing Date: %s", reportTime.Format("02 January 2006")))
	pdf.Ln(lineBreakSize * 1.5)

	currentY := pdf.GetY()

	chartBuf, err := drawChart(bypassesNum, blockedNum, bypassesNum+blockedNum, "Bypassed", "Blocked", "Detection Score")
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

	chartFalseBuf, err := drawChart(truePosNum, falsePosNum, truePosNum+falsePosNum, "True Positive", "False Positive", "Positive Tests Score")
	if err == nil {
		imageInfoFalse := pdf.RegisterImageReader("False Pos Plot", "PNG", chartFalseBuf)
		if pdf.Ok() {
			imgWd, imgHt := imageInfoFalse.Extent()
			imgWd, imgHt = imgWd/2, imgHt/2
			pdf.Image("False Pos Plot", pageWidth-imgWd-pageWidth/20, currentY,
				imgWd, imgHt, true, "PNG", 0, "")
		}
	}

	// Num of bypasses: failed tests minus positive cases minus unknown cases
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("%v bypasses in %v tests, %v unresolved cases / %v test cases",
		len(maliciousRows)-1, db.overallTestsCompleted, len(db.naTests), db.overallTestcasesCompleted))
	pdf.Ln(lineBreakSize)

	tableClip(pdf, cols, rows, 10)

	httpimg.Register(pdf, trollLink, "")
	pdf.Image(trollLink, 15, 280, 20, 0, false, "", 0, wallarmLink)

	// Positive tests page
	pdf.AddPage()
	pdf.SetFont("Arial", "", 24)
	pdf.Cell(cellWidth, cellHeight, "Positive Tests in Details")
	pdf.Ln(lineBreakSize)

	// False Positive payloads block
	cols = []float64{100, 30, 20, 25, 15}

	pdf.SetFont("Arial", "", 12)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("\n%d false positive requests identified as blocked (failed, bad behavior)", len(falsePosRows)-1))
	pdf.Ln(lineBreakSize)
	pdf.SetFont("Arial", "", 10)

	tableClip(pdf, cols, falsePosRows, 10)

	// True Positive payloads block
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("\n%d true positive requests identified as bypassed (passed, good behavior)", len(truePosRows)-1))
	pdf.Ln(lineBreakSize)
	pdf.SetFont("Arial", "", 10)

	tableClip(pdf, cols, truePosRows, 10)

	// Malicious payloads page
	pdf.AddPage()
	pdf.SetFont("Arial", "", 24)
	pdf.Cell(cellWidth, cellHeight, "Bypasses in Details")
	pdf.Ln(lineBreakSize)

	pdf.SetFont("Arial", "", 12)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("\n%d malicious requests have bypassed the WAF", len(maliciousRows)-1))
	pdf.Ln(lineBreakSize)

	pdf.SetFont("Arial", "", 10)
	tableClip(pdf, cols, maliciousRows, 10)

	cols = []float64{100, 30, 20, 25, 15}
	var unresolvedRaws [][]string
	unresolvedRaws = append(unresolvedRaws, []string{"Payload", "Test Case", "Encoder", "Placeholder", "Status"})
	for _, naTest := range db.naTests {
		payload := fmt.Sprintf("%+q", naTest.Payload)
		payload = strings.ReplaceAll(payload[1:len(payload)-1], `\"`, `"`)
		unresolvedRaws = append(unresolvedRaws,
			[]string{payload,
				naTest.Case,
				naTest.Encoder,
				naTest.Placeholder,
				strconv.Itoa(naTest.ResponseStatusCode)},
		)
	}

	pdf.AddPage()
	pdf.SetFont("Arial", "", 24)
	pdf.Cell(cellWidth, cellHeight, "Unresolved Test Cases")
	pdf.Ln(lineBreakSize)
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("\n%d requests indentified as blocked and passed or as not-blocked and not-passed",
		len(db.naTests)))
	pdf.Ln(lineBreakSize)
	pdf.SetFont("Arial", "", 10)

	tableClip(pdf, cols, unresolvedRaws, 10)

	err = pdf.OutputFileAndClose(reportFile)
	if err != nil {
		return errors.Wrap(err, "PDF generation error")
	}

	fmt.Printf("\nPDF report is ready: %s\n", reportFile)
	return nil
}
