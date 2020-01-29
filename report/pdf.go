package report

import (
	"fmt"

	"github.com/jung-kurt/gofpdf"
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
	//TODO: implement a real PDF document creation here
	cols := []float64{25, 25, 25, 25}
	rows := [][]string{}
	//for i := 1; i <= 88; i++ {
	//	word := fmt.Sprintf("%d:%s", i, strings.Repeat("A", i))
	//	rows = append(rows, []string{word, word, word})
	//}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Arial", "", 12)
	pdf.AddPage()

	for testset := range r.Report {
		for testcase := range r.Report[testset] {
			passed := r.Report[testset][testcase][true]
			failed := r.Report[testset][testcase][false]
			total := passed + failed
			percentage := float32(passed) / float32(total)
			rows = append(rows, []string{testset, testcase, fmt.Sprintf("%d", passed), fmt.Sprintf("%d", failed)})
			fmt.Printf("%v\t%v\t%v/%v\t(%.2f)", testset, testcase, passed, total, percentage)
		}
	}

	tableClip(pdf, cols, rows)
	pdf.OutputFileAndClose("tables.pdf")
}
