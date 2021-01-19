package report

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"os"
	"strconv"

	"github.com/jung-kurt/gofpdf"
	"github.com/jung-kurt/gofpdf/contrib/httpimg"
	"github.com/pkg/errors"
)

const MARGECELL = 2 // marge top/bottom of cell

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
		// add a new page if the height of the row doesn't fit on the page
		if y+height >= pageh-mbottom {
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
			pdf.Rect(x, y, width, height, "")
			pdf.ClipRect(x, y, width, height, false)
			pdf.Cell(width, height, txt)
			pdf.ClipEnd()
			x += width
		}
		pdf.Ln(-1)
	}
}

func (r *Report) ExportToPDFAndShowTable(reportFile string) error {
	// Process data.
	var rows [][]string
	overallPassedRate := float32(0)
	overallTestsCompleted := 0
	overallTestsFailed := 0
	overallTestcasesCompleted := float32(0)

	rows = append(rows, []string{"Test set", "Test case", "Passed, %", "Passed/Blocked", "Failed/Bypassed"})

	for testSet := range r.Report {
		for testCase := range r.Report[testSet] {
			passed := r.Report[testSet][testCase][true]
			failed := r.Report[testSet][testCase][false]
			total := passed + failed
			overallTestsCompleted += total
			overallTestsFailed += failed
			percentage := float32(passed) / float32(total) * 100
			rows = append(rows, []string{testSet, testCase, fmt.Sprintf("%.2f", percentage), fmt.Sprintf("%d", passed), fmt.Sprintf("%d", failed)})
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
	cols := []float64{35, 35, 35, 35, 35, 35}
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "", 24)
	pdf.Cell(10, 10, fmt.Sprintf("WAF score: %.2f%%", wafScore))

	pdf.Ln(10)
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(10, 10, fmt.Sprintf("%v bypasses in %v tests / %v test cases", overallTestsFailed, overallTestsCompleted, overallTestcasesCompleted))
	pdf.Ln(10)

	tableClip(pdf, cols, rows, 12)

	url := "http://troll.wallarm.tools/assets/wallarm.logo.png"
	httpimg.Register(pdf, url, "")
	pdf.Image(url, 15, 280, 20, 0, false, "", 0, "https://wallarm.com/?utm_campaign=gtw_tool&utm_medium=pdf&utm_source=github")

	pdf.AddPage()

	cols = []float64{135, 20, 20, 15}
	rows = [][]string{}

	rows = append(rows, []string{"Payload", "Encoder", "Placeholder", "Status"})
	for _, failedTest := range r.FailedTests {
		rows = append(rows, []string{failedTest.Payload, failedTest.Encoder, failedTest.Placeholder, strconv.Itoa(failedTest.StatusCode)})
	}
	pdf.SetFont("Arial", "", 24)
	pdf.Cell(10, 10, "Bypasses in details.")
	pdf.Ln(10)
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(10, 10, fmt.Sprintf("\n%d malicious requests bypassed the WAF", len(r.FailedTests)))
	pdf.Ln(10)
	pdf.SetFont("Arial", "", 10)

	tableClip(pdf, cols, rows, 10)

	pdf.AddPage()

	cols = []float64{135, 20, 20, 15}
	rows = [][]string{}

	rows = append(rows, []string{"Payload", "Encoder", "Placeholder", "Status"})
	for _, naTest := range r.NaTests {
		rows = append(rows, []string{naTest.Payload, naTest.Encoder, naTest.Placeholder, strconv.Itoa(naTest.StatusCode)})
	}
	pdf.SetFont("Arial", "", 24)
	pdf.Cell(10, 10, "Unresolved test cases")
	pdf.Ln(10)
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(10, 10, fmt.Sprintf("\n%d requests indentified as blocked and passed both or as not-blocked and not-passed both", len(r.NaTests)))
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
