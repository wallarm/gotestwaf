package test

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/jung-kurt/gofpdf/contrib/httpimg"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
)

const (
	MARGECELL = 2 // marge top/bottom of cell

	wallarmLink = "https://wallarm.com/?utm_campaign=gtw_tool&utm_medium=pdf&utm_source=github"
	trollLink   = "http://troll.wallarm.tools/assets/wallarm.logo.png"

	cellWidth     = 10
	cellHeight    = 10
	lineBreakSize = 10
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
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("%v bypasses in %v tests / %v test cases",
		overallTestsFailed, overallTestsCompleted, overallTestcasesCompleted))
	pdf.Ln(lineBreakSize)

	tableClip(pdf, cols, rows, 12)

	httpimg.Register(pdf, trollLink, "")
	pdf.Image(trollLink, 15, 280, 20, 0, false, "", 0, wallarmLink)

	pdf.AddPage()

	cols = []float64{100, 30, 20, 25, 15}
	rows = [][]string{}

	rows = append(rows, []string{"Payload", "Test Case", "Encoder", "Placeholder", "Status"})
	for _, failedTest := range db.failedTests {
		payload := fmt.Sprintf("%+q", failedTest.Payload)
		payload = strings.ReplaceAll(payload[1:len(payload)-1], `\"`, `"`)
		rows = append(rows,
			[]string{payload,
				failedTest.Case,
				failedTest.Encoder,
				failedTest.Placeholder,
				strconv.Itoa(failedTest.ResponseStatusCode)},
		)
	}
	pdf.SetFont("Arial", "", 24)
	pdf.Cell(cellWidth, cellHeight, "Bypasses in details.")
	pdf.Ln(lineBreakSize)
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("\n%d malicious requests have bypassed the WAF", len(db.failedTests)))
	pdf.Ln(lineBreakSize)
	pdf.SetFont("Arial", "", 10)

	tableClip(pdf, cols, rows, 10)

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
	pdf.Cell(cellWidth, cellHeight, "Unresolved test cases")
	pdf.Ln(lineBreakSize)
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(cellWidth, cellHeight, fmt.Sprintf("\n%d requests indentified as blocked and passed or as not-blocked and not-passed",
		len(db.naTests)))
	pdf.Ln(lineBreakSize)
	pdf.SetFont("Arial", "", 10)

	tableClip(pdf, cols, rows, 10)

	err := pdf.OutputFileAndClose(reportFile)
	if err != nil {
		return errors.Wrap(err, "PDF generation error")
	}

	fmt.Printf("\nPDF report is ready: %s\n", reportFile)
	return nil
}
