package test

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"

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

func drawChart(bypassed, blocked, overall int, failed, passed, title string) (*bytes.Buffer, error) {
	bypassedPercentage := float64(bypassed*100) / float64(overall)
	blockedPercentage := 100.0 - bypassedPercentage
	pieChartImg := chart.PieChart{
		DPI:   85,
		Title: title,
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
				Label: fmt.Sprintf("%s: %d (%.2f%%)", failed, bypassed, bypassedPercentage),
				Style: chart.Style{
					// Red
					FillColor: drawing.ColorFromAlphaMixedRGBA(66, 133, 244, 255),
					FontSize:  12,
				},
			},
			{
				Value: float64(blocked),
				Label: fmt.Sprintf("%s: %d (%.2f%%)", passed, blocked, blockedPercentage),
				Style: chart.Style{
					// Blue
					FillColor: drawing.ColorFromAlphaMixedRGBA(234, 67, 54, 255),
					FontSize:  12,
				},
			},
		},
	}
	buffer := bytes.NewBuffer([]byte{})
	if err := pieChartImg.Render(chart.PNG, buffer); err != nil {
		return buffer, err
	}
	return buffer, nil
}

func calculatePercentage(first, second int) float32 {
	if second == 0 {
		return 0.0
	}
	return float32(first) / float32(second) * 100
}

func isPositiveTest(setName string) bool {
	return strings.Contains(setName, "false")
}

func (db *DB) RenderTable(reportTime time.Time, wafName string, ignoreUnresolved bool) ([][]string, error) {
	baseHeader := []string{"Test set", "Test case", "Percentage, %", "Blocked", "Bypassed"}
	if !ignoreUnresolved {
		baseHeader = append(baseHeader, "Unresolved")
	}

	// Table rows to render, regular and positive cases
	positiveTestRows := [][]string{baseHeader}
	negativeTestRows := [][]string{baseHeader}

	// Counters to use with table footers
	positiveRequestsNumber := make(map[string]int)
	negativeRequestsNumber := make(map[string]int)
	unresolvedRequestsNumber := make(map[string]int)

	var positiveUnresolvedRequestsSum int
	for _, unresolvedTest := range db.naTests {
		// If we want to count UNRESOLVED as BYPASSED, we shouldn't count UNRESOLVED at all
		// set it to zero by default
		if ignoreUnresolved {
			unresolvedRequestsNumber[unresolvedTest.Case] = 0
			continue
		}
		if isPositiveTest(unresolvedTest.Set) {
			positiveUnresolvedRequestsSum++
		}
		unresolvedRequestsNumber[unresolvedTest.Case]++
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
			unresolvedRequests := unresolvedRequestsNumber[testCase]
			passedRequests := db.counters[testSet][testCase][true]
			failedRequests := db.counters[testSet][testCase][false]
			totalRequests := passedRequests + failedRequests
			// If we don't want to count UNRESOLVED requests as BYPASSED, we need to subtract them
			// from failed requests (in other case we will count them as usual), and add this
			// subtracted value to the overall requests
			if !ignoreUnresolved {
				failedRequests -= unresolvedRequests
			}

			totalResolvedRequests := passedRequests + failedRequests
			var passedRequestsPercentage float32 = 0
			if totalResolvedRequests != 0 {
				passedRequestsPercentage = float32(passedRequests) / float32(totalResolvedRequests) * 100
			}

			db.overallRequests += totalRequests
			db.overallRequestsFailed += failedRequests

			// If positive set - move to another table (remove from general cases)
			if isPositiveTest(testSet) {
				// False positive - blocked by the WAF (bad behavior, failedRequests)
				positiveRequestsNumber["blocked"] += failedRequests
				// True positive - bypassed (good behavior, passedRequests)
				positiveRequestsNumber["bypassed"] += passedRequests

				// Swap the "failedRequests" and "passedRequests" cases for positive cases
				rowAppend := []string{
					testSet,
					testCase,
					fmt.Sprintf("%.2f", passedRequestsPercentage),
					fmt.Sprintf("%d", failedRequests),
					fmt.Sprintf("%d", passedRequests)}
				if !ignoreUnresolved {
					rowAppend = append(rowAppend, fmt.Sprintf("%d", unresolvedRequestsNumber[testCase]))
				}

				positiveTestRows = append(positiveTestRows, rowAppend)
				continue
			}

			// If not positive set - insert into the original table, update stats
			rowAppend := []string{
				testSet,
				testCase,
				fmt.Sprintf("%.2f", passedRequestsPercentage),
				fmt.Sprintf("%d", passedRequests),
				fmt.Sprintf("%d", failedRequests)}
			if !ignoreUnresolved {
				rowAppend = append(rowAppend, fmt.Sprintf("%d", unresolvedRequestsNumber[testCase]))
			}

			negativeRequestsNumber["blocked"] += passedRequests
			negativeRequestsNumber["bypassed"] += failedRequests

			negativeTestRows = append(negativeTestRows, rowAppend)

			db.overallCompletedTestCases += 1.00
			db.overallPassedRequestsPercentage += passedRequestsPercentage
		}
	}

	db.wafScore = db.overallPassedRequestsPercentage / db.overallCompletedTestCases

	// Create a table for regular cases (excluding positive cases)
	fmt.Println("\nNegative Tests:")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(baseHeader)

	for _, row := range negativeTestRows[1:] {
		table.Append(row)
	}
	for index := range baseHeader {
		table.SetColMinWidth(index, colMinWidth)
	}

	positiveRequestsSum := positiveRequestsNumber["blocked"] + positiveRequestsNumber["bypassed"] + positiveUnresolvedRequestsSum
	negativeRequestsSum := db.overallRequests - positiveRequestsSum

	positiveResolvedRequestsSum := positiveRequestsSum - positiveUnresolvedRequestsSum
	negativeResolvedRequestsSum := db.overallRequests - positiveResolvedRequestsSum
	if !ignoreUnresolved {
		negativeResolvedRequestsSum -= len(db.naTests)
	}

	var negativeUnresolvedRequestsSum int
	if !ignoreUnresolved {
		negativeUnresolvedRequestsSum = len(db.naTests) - positiveUnresolvedRequestsSum
	}

	negativeUnresolvedRequestsPercentage := calculatePercentage(negativeUnresolvedRequestsSum, negativeRequestsSum)
	negativeResolvedBlockedRequestsPercentage := calculatePercentage(negativeRequestsNumber["blocked"], negativeResolvedRequestsSum)
	negativeResolvedBypassedRequestsPercentage := calculatePercentage(negativeRequestsNumber["bypassed"], negativeResolvedRequestsSum)

	footerNegativeTests := []string{
		fmt.Sprintf("Date:\n%s", reportTime.Format("2006-01-02")),
		fmt.Sprintf("WAF Name:\n%s", wafName),
		fmt.Sprintf("WAF Average Score:\n%.2f%%", db.wafScore),
		fmt.Sprintf("Blocked (Resolved):\n%d/%d (%.2f%%)", negativeRequestsNumber["blocked"], negativeResolvedRequestsSum, negativeResolvedBlockedRequestsPercentage),
		fmt.Sprintf("Bypassed (Resolved):\n%d/%d (%.2f%%)", negativeRequestsNumber["bypassed"], negativeResolvedRequestsSum, negativeResolvedBypassedRequestsPercentage)}
	if !ignoreUnresolved {
		footerNegativeTests = append(footerNegativeTests, fmt.Sprintf("Unresolved:\n%d/%d (%.2f%%)", negativeUnresolvedRequestsSum, negativeRequestsSum, negativeUnresolvedRequestsPercentage))
	}
	table.SetFooter(footerNegativeTests)
	table.Render()

	// Create a table for positive cases
	fmt.Println("\nPositive Tests:")
	posTable := tablewriter.NewWriter(os.Stdout)
	posTable.SetHeader(baseHeader)

	for _, row := range positiveTestRows[1:] {
		posTable.Append(row)
	}
	for index := range baseHeader {
		posTable.SetColMinWidth(index, colMinWidth)
	}

	positiveUnresolvedRequestsPercentage := calculatePercentage(positiveUnresolvedRequestsSum, positiveRequestsSum)
	positiveResolvedFalsePercentage := calculatePercentage(positiveRequestsNumber["blocked"], positiveResolvedRequestsSum)
	positiveResolvedTruePercentage := calculatePercentage(positiveRequestsNumber["bypassed"], positiveResolvedRequestsSum)

	footerPositiveTests := []string{
		fmt.Sprintf("Date:\n%s", reportTime.Format("2006-01-02")),
		fmt.Sprintf("WAF Name:\n%s", wafName),
		fmt.Sprintf("WAF Positive Score:\n%.2f%%", positiveResolvedTruePercentage),
		fmt.Sprintf("False positive (res):\n%d/%d (%.2f%%)", positiveRequestsNumber["blocked"], positiveResolvedRequestsSum, positiveResolvedFalsePercentage),
		fmt.Sprintf("True positive (res):\n%d/%d (%.2f%%)", positiveRequestsNumber["bypassed"], positiveResolvedRequestsSum, positiveResolvedTruePercentage)}
	if !ignoreUnresolved {
		footerPositiveTests = append(footerPositiveTests, fmt.Sprintf("Unresolved:\n%d/%d (%.2f%%)", positiveUnresolvedRequestsSum, positiveRequestsSum, positiveUnresolvedRequestsPercentage))
	}

	posTable.SetFooter(footerPositiveTests)
	posTable.Render()

	return negativeTestRows, nil
}

func (db *DB) ExportToPDF(reportFile string, reportTime time.Time, wafName, url string, rows [][]string, ignoreUnresolved bool) error {
	baseHeader := []string{"Payload", "Test Case", "Encoder", "Placeholder", "Status"}

	negativeBypassRows := [][]string{baseHeader}
	positiveTrueRows := [][]string{baseHeader}
	positiveFalseRows := [][]string{baseHeader}

	for _, failedRequest := range db.failedTests {
		payload := fmt.Sprintf("%+q", failedRequest.Payload)
		payload = strings.ReplaceAll(payload[1:len(payload)-1], `\"`, `"`)
		toAppend := []string{payload,
			failedRequest.Case,
			failedRequest.Encoder,
			failedRequest.Placeholder,
			strconv.Itoa(failedRequest.ResponseStatusCode)}
		if isPositiveTest(failedRequest.Set) {
			positiveFalseRows = append(positiveFalseRows, toAppend)
		} else {
			negativeBypassRows = append(negativeBypassRows, toAppend)
		}
	}

	for _, passedRequest := range db.passedTests {
		payload := fmt.Sprintf("%+q", passedRequest.Payload)
		payload = strings.ReplaceAll(payload[1:len(payload)-1], `\"`, `"`)
		// Passed for false pos - bypassed (good behavior)
		if isPositiveTest(passedRequest.Set) {
			positiveTrueRows = append(positiveTrueRows, []string{payload,
				passedRequest.Case,
				passedRequest.Encoder,
				passedRequest.Placeholder,
				strconv.Itoa(passedRequest.ResponseStatusCode)})
		}
	}

	// Num (general): number of actual rows minus top header (1 line)
	positiveTrueNumber := len(positiveTrueRows) - 1
	positiveFalseNumber := len(positiveFalseRows) - 1
	var positiveUnresolvedNumber int
	for _, unresolvedRequest := range db.naTests {
		if isPositiveTest(unresolvedRequest.Set) {
			positiveUnresolvedNumber++
		}
	}

	if ignoreUnresolved {
		for _, unresolvedRequest := range db.naTests {
			payload := fmt.Sprintf("%+q", unresolvedRequest.Payload)
			payload = strings.ReplaceAll(payload[1:len(payload)-1], `\"`, `"`)
			negativeBypassRows = append(negativeBypassRows, []string{payload,
				unresolvedRequest.Case,
				unresolvedRequest.Encoder,
				unresolvedRequest.Placeholder,
				strconv.Itoa(unresolvedRequest.ResponseStatusCode)})
		}
	}

	negativeBypassNumber := len(negativeBypassRows) - 1
	negativeBlockedNumber := len(db.passedTests) - positiveTrueNumber

	columns := []float64{25, 35, 30, 35, 35, 30}

	// Title page
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.SetFont("Arial", "", 24)
	pdf.Cell(cellWidth, cellHeight, "WAF Testing Results")
	pdf.Ln(lineBreakSize)

	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("WAF Average Score: %.2f%%", db.wafScore))
	pdf.Ln(lineBreakSize / 2)

	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("WAF Detection Score: %.2f%%", calculatePercentage(negativeBlockedNumber, negativeBypassNumber+negativeBlockedNumber)))
	pdf.Ln(lineBreakSize / 2)

	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("WAF Positive Tests Score: %.2f%%", calculatePercentage(positiveTrueNumber, positiveTrueNumber+positiveFalseNumber)))
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

	negativeChartFlow := false
	// Show only negative chart if positive chart is not available
	if positiveTrueNumber+positiveFalseNumber == 0 {
		negativeChartFlow = true
	}

	if negativeBypassNumber+negativeBlockedNumber != 0 {
		chartBuf, err := drawChart(negativeBypassNumber, negativeBlockedNumber, negativeBypassNumber+negativeBlockedNumber, "Bypassed", "Blocked", "Detection Score")
		if err != nil {
			return errors.Wrap(err, "Plot generation error (negative tests)")
		}
		imageInfo := pdf.RegisterImageReader("Overall Plot", "PNG", chartBuf)
		if pdf.Ok() {
			imgWd, imgHt := imageInfo.Extent()
			imgWd, imgHt = imgWd/2, imgHt/2
			pdf.Image("Overall Plot", pageWidth/20, currentY,
				imgWd, imgHt, negativeChartFlow, "PNG", 0, "")
		}
	}
	if positiveTrueNumber+positiveFalseNumber != 0 {
		chartFalseBuf, err := drawChart(positiveTrueNumber, positiveFalseNumber, positiveTrueNumber+positiveFalseNumber, "True Positive", "False Positive", "Positive Tests Score")
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

	// Num of bypasses: failed tests minus positive cases minus unknown cases
	unresolvedRequests := db.overallRequests - negativeBypassNumber - negativeBlockedNumber - positiveTrueNumber - positiveFalseNumber
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("Total: %v bypasses in %v tests, %v unresolved cases / %v test cases",
		negativeBypassNumber, db.overallRequests, unresolvedRequests, db.overallCompletedTestCases))
	pdf.Ln(lineBreakSize)

	tableClip(pdf, columns, rows, 10)

	reader := bytes.NewReader(resources.WallarmLogo)
	pdf.RegisterImageReader("wallarm-logo", "PNG", reader)
	pdf.Image("wallarm-logo", 15, 280, 20, 0, false, "", 0, wallarmLink)

	// Positive tests page
	pdf.AddPage()
	pdf.SetFont("Arial", "", 24)
	pdf.Cell(cellWidth, cellHeight, "Positive Tests in Details")
	pdf.Ln(lineBreakSize)

	// False Positive payloads block
	columns = []float64{100, 30, 20, 25, 15}

	pdf.SetFont("Arial", "", 12)
	pdf.Cell(
		cellWidth,
		cellHeight,
		fmt.Sprintf("\n%d false positive requests identified as blocked (failed, bad behavior)",
			len(positiveFalseRows)-1),
	)
	pdf.Ln(lineBreakSize)
	pdf.SetFont("Arial", "", 10)

	tableClip(pdf, columns, positiveFalseRows, 10)

	// True Positive payloads block
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(
		cellWidth,
		cellHeight,
		fmt.Sprintf("\n%d true positive requests identified as bypassed (passed, good behavior)",
			len(positiveTrueRows)-1),
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
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("\n%d malicious requests have bypassed the WAF", len(negativeBypassRows)-1))
	pdf.Ln(lineBreakSize)

	pdf.SetFont("Arial", "", 10)
	tableClip(pdf, columns, negativeBypassRows, 10)

	columns = []float64{100, 30, 20, 25, 15}
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

	if !ignoreUnresolved {
		pdf.AddPage()
		pdf.SetFont("Arial", "", 24)
		pdf.Cell(cellWidth, cellHeight, "Unresolved Test Cases")
		pdf.Ln(lineBreakSize)
		pdf.SetFont("Arial", "", 12)
		pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("\n%d requests indentified as blocked and passed or as not-blocked and not-passed",
			len(db.naTests)))
		pdf.Ln(lineBreakSize)
		pdf.SetFont("Arial", "", 10)

		tableClip(pdf, columns, unresolvedRaws, 10)
	}

	if err := pdf.OutputFileAndClose(reportFile); err != nil {
		return errors.Wrap(err, "PDF generation error")
	}

	fmt.Printf("\nPDF report is ready: %s\n", reportFile)
	return nil
}
