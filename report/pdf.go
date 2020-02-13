package report

import (
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/jung-kurt/gofpdf/contrib/httpimg"
)

const MARGECELL = 2 // marge top/bottom of cell

func tableClip(pdf *gofpdf.Fpdf, cols []float64, rows [][]string) {
	pagew, pageh := pdf.GetPageSize()
	_ = pagew
	mleft, mright, mtop, mbottom := pdf.GetMargins()
	_ = mleft
	_ = mright
	_ = mtop

	for _, row := range rows {
		_, lineHt := pdf.GetFontSize()
		height := lineHt + MARGECELL

		x, y := pdf.GetXY()
		// add a new page if the height of the row doesn't fit on the page
		if y+height >= pageh-mbottom {
			pdf.AddPage()
			x, y = pdf.GetXY()
		}
		for i, txt := range row {
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

func (r Report) ExportPDF() {
	cols := []float64{35, 35, 35, 35, 35, 35}
	rows := [][]string{}
	overallPassedRate := float32(0)
	overallTestsCompleted := 0
	overallTestsFailed := 0
	overallTestcasesCompleted := float32(0)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	rows = append(rows, []string{"Test set", "Test case", "Passed, %", "Passed/Blocked", "Failed/Bypassed"})

	for testset := range r.Report {
		for testcase := range r.Report[testset] {
			passed := r.Report[testset][testcase][true]
			failed := r.Report[testset][testcase][false]
			total := passed + failed
			overallTestsCompleted += total
			overallTestsFailed += failed
			percentage := float32(passed) / float32(total)
			rows = append(rows, []string{testset, testcase, fmt.Sprintf("%.2f", percentage), fmt.Sprintf("%d", passed), fmt.Sprintf("%d", failed)})
			fmt.Printf("%v\t%v\t%v/%v\t(%.2f)\n", testset, testcase, passed, total, percentage)
			overallTestcasesCompleted += 1.00
			overallPassedRate += percentage
		}
	}

	pdf.SetFont("Arial", "", 24)
	pdf.Cell(10, 10, fmt.Sprintf("WAF score: %.2f%%", (overallPassedRate/overallTestcasesCompleted)*100))
	pdf.Ln(10)
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(10, 10, fmt.Sprintf("%v bypasses in %v tests / %v test cases", overallTestsFailed, overallTestsCompleted, overallTestcasesCompleted))
	pdf.Ln(10)

	tableClip(pdf, cols, rows)

	url := "http://troll.wallarm.tools/assets/wallarm.logo.png"
	httpimg.Register(pdf, url, "")
	pdf.Image(url, 15, 280, 20, 0, false, "", 0, "https://wallarm.com/?utm_campaign=gtw_tool&utm_medium=pdf&utm_source=github")

	//pdf.Image(url, 15, 15, 510, 0, false, "", 0, "")
	current := time.Now()
	pdf.OutputFileAndClose("/tmp/report/waf-test-report" + current.Format("2019-12-31") + ".pdf")
}
