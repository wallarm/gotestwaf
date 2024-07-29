package report

import (
	"context"
	"os"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
)

// chromium-browser \
// --headless \
// --no-zygote \
// --single-process \
// --no-sandbox \
// --disable-gpu \
// --run-all-compositor-stages-before-draw \
// --no-pdf-header-footer \
// --print-to-pdf=test.pdf \
// report.html

var chromeDPExecAllocatorOptions = append(
	chromedp.DefaultExecAllocatorOptions[:],
	chromedp.Flag("no-zygote", true),
	chromedp.Flag("no-sandbox", true),
	chromedp.Flag("disable-gpu", true),
	chromedp.Flag("run-all-compositor-stages-before-draw", true),
)

func renderToPDF(ctx context.Context, fileToRenderURL string, pathToResultPDF string) error {
	allocCtx, allocCtxCancel := chromedp.NewExecAllocator(ctx, chromeDPExecAllocatorOptions...)
	defer allocCtxCancel()

	chromeCtx, chromeCtxCancel := chromedp.NewContext(allocCtx)
	defer chromeCtxCancel()

	var buf []byte

	tasks := chromedp.Tasks{
		chromedp.Navigate(fileToRenderURL),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			buf, _, err = page.PrintToPDF().WithPrintBackground(true).Do(ctx)
			if err != nil {
				return err
			}
			return nil
		}),
	}

	if err := chromedp.Run(chromeCtx, tasks); err != nil {
		return errors.Wrap(err, "couldn't render HTML file to PDF")
	}

	if err := os.WriteFile(pathToResultPDF, buf, 0o644); err != nil {
		return errors.Wrap(err, "couldn't save PDF file")
	}

	return nil
}
