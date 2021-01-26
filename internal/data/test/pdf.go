package test

import (
	"fmt"
	"sort"

	"os"
	"strconv"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"github.com/jung-kurt/gofpdf/contrib/httpimg"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
)

const (
	MARGECELL = 2 // marge top/bottom of cell

	wallarmLink = "https://wallarm.com/?utm_campaign=gtw_tool&utm_medium=pdf&utm_source=github"
	trollLink   = "http://troll.wallarm.tools/assets/wallarm.logo.png"
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

		//found max number of lines in the cell to create one size cells in the row
		var nLines = make([]int, len(row))
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
		pdf.Ln(height*float64(maxNLine))
	}
}

func (db *DB) ExportToPDFAndShowTable(reportFile string) error {
	// Process data.
	var rows [][]string
	overallPassedRate := float32(0)
	overallTestsCompleted := 0
	overallTestsFailed := 0
	overallTestcasesCompleted := float32(0)

	rows = append(rows, []string{"Test set", "Test case", "Passed, %", "Passed/Blocked", "Failed/Bypassed"})

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
			percentage := float32(passed) / float32(total) * 100
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
	table.SetHeader([]string{"Test Set", "Test Case", "Passed", "Blocked", "Failed/Bypassed"})
	table.SetFooter([]string{"", "", "", "WAF Score:", fmt.Sprintf("%.2f%%", wafScore)})

	for _, v := range rows[1:] {
		table.Append(v)
	}
	table.Render()

	// Create a pdf file
	cols := []float64{35, 45, 35, 35, 40}
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "", 24)
	pdf.Cell(10, 10, fmt.Sprintf("WAF score: %.2f%%", wafScore))

	pdf.Ln(10)
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(10, 10, fmt.Sprintf("%v bypasses in %v tests / %v test cases",
		overallTestsFailed, overallTestsCompleted, overallTestcasesCompleted))
	pdf.Ln(10)

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
		rows = append(rows, []string{payload, failedTest.TestCase, failedTest.Encoder, failedTest.Placeholder, strconv.Itoa(failedTest.StatusCode)})
	}
	pdf.SetFont("Arial", "", 24)
	pdf.Cell(10, 10, "Bypasses in details.")
	pdf.Ln(10)
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(10, 10, fmt.Sprintf("\n%d malicious requests have bypassed the WAF", len(db.failedTests)))
	pdf.Ln(10)
	pdf.SetFont("Arial", "", 10)

	tableClip(pdf, cols, rows, 10)

	pdf.AddPage()

	cols = []float64{100, 30, 20, 25, 15}
	rows = [][]string{}

	rows = append(rows, []string{"Payload", "Test Case", "Encoder", "Placeholder", "Status"})
	for _, naTest := range db.naTests {
		payload := fmt.Sprintf("%+q", naTest.Payload)
		payload = strings.ReplaceAll(payload[1:len(payload)-1], `\"`, `"`)
		rows = append(rows, []string{payload, naTest.TestCase, naTest.Encoder, naTest.Placeholder, strconv.Itoa(naTest.StatusCode)})
	}
	pdf.SetFont("Arial", "", 24)
	pdf.Cell(10, 10, "Unresolved test cases")
	pdf.Ln(10)
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(10, 10, fmt.Sprintf("\n%d requests indentified as blocked and passed or as not-blocked and not-passed",
		len(db.naTests)))
	pdf.Ln(10)
	pdf.SetFont("Arial", "", 10)

	tableClip(pdf, cols, rows, 10)

	err := pdf.OutputFileAndClose(reportFile)
	if err != nil {
		return errors.Wrap(err, "PDF generation error")
	}

	fmt.Printf("\nPDF report is ready: %s\n", reportFile)
	return nil
}
