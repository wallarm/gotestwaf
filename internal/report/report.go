package report

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/db"
)

const (
	maxReportFilenameLength = 249 // 255 (max length) - 5 (".html") - 1 (to be sure)

	consoleReportTextFormat = "text"
	consoleReportJsonFormat = "json"

	JsonFormat = "json"
	HtmlFormat = "html"
	PdfFormat  = "pdf"
	NoneFormat = "none"
)

func SendReportByEmail(
	ctx context.Context, s *db.Statistics, email string, reportTime time.Time,
	wafName string, url string, openApiFile string, args []string, ignoreUnresolved bool, includePayloads bool,
) error {
	reportData, err := oncePrepareHTMLFullReport(s, reportTime, wafName, url, openApiFile, args, ignoreUnresolved, includePayloads)
	if err != nil {
		return errors.Wrap(err, "couldn't prepare data for HTML report")
	}

	err = sendEmail(ctx, reportData, email)
	if err != nil {
		return err
	}

	return nil
}

// ExportFullReport saves full report on disk in different formats: HTML, PDF, JSON.
func ExportFullReport(
	ctx context.Context, s *db.Statistics, reportFile string, reportTime time.Time,
	wafName string, url string, openApiFile string, args []string, ignoreUnresolved bool,
	includePayloads bool, format string,
) (fullName string, err error) {
	_, reportFileName := filepath.Split(reportFile)
	if len(reportFileName) > maxReportFilenameLength {
		return "", errors.New("report filename too long")
	}

	switch format {
	case HtmlFormat:
		fullName = reportFile + ".html"
		err = printFullReportToHtml(s, fullName, reportTime, wafName, url, openApiFile, args, ignoreUnresolved, includePayloads)
		if err != nil {
			return "", err
		}

	case PdfFormat:
		fullName = reportFile + ".pdf"
		err = printFullReportToPdf(ctx, s, fullName, reportTime, wafName, url, openApiFile, args, ignoreUnresolved, includePayloads)
		if err != nil {
			return "", err
		}

	case JsonFormat:
		fullName = reportFile + ".json"
		err = printFullReportToJson(s, fullName, reportTime, wafName, url, args, ignoreUnresolved)
		if err != nil {
			return "", err
		}

	case NoneFormat:
		return "", nil

	default:
		return "", fmt.Errorf("unknown report format: %s", format)
	}

	return fullName, nil
}
