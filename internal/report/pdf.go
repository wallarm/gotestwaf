package report

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/db"
)

func printFullReportToPdf(
	ctx context.Context, s *db.Statistics, reportFile string, reportTime time.Time,
	wafName string, url string, openApiFile string, args []string, ignoreUnresolved bool,
	includePayloads bool,
) error {
	tempFileName, err := exportFullReportToHtml(s, reportTime, wafName, url, openApiFile, args, ignoreUnresolved, includePayloads)
	if err != nil {
		return errors.Wrap(err, "couldn't export report to HTML")
	}

	err = renderToPDF(ctx, tempFileName, reportFile)
	if err != nil {
		return errors.Wrap(err, "couldn't render HTML report to PDF")
	}

	return nil
}
