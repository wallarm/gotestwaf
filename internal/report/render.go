package report

import (
	"context"
	"os"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
)

func renderToPDF(ctx context.Context, fileToRenderURL string, pathToResultPDF string) error {
	chromeCtx, cancel := chromedp.NewContext(ctx)
	defer cancel()

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
