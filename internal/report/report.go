package report

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/db"
)

const (
	maxReportFilenameLength = 249 // 255 (max length) - 5 (".html") - 1 (to be sure)

	consoleReportTextFormat = "text"
	consoleReportJsonFormat = "json"
)
const (
	NoneFormat = "none"
	JsonFormat = "json"
	HtmlFormat = "html"
	PdfFormat  = "pdf"
)

var (
	ReportFormatsSet = map[string]any{
		NoneFormat: nil,
		JsonFormat: nil,
		HtmlFormat: nil,
		PdfFormat:  nil,
	}
	ReportFormats = slices.Collect(maps.Keys(ReportFormatsSet))
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
	includePayloads bool, formats []string,
) (reportFileNames []string, err error) {
	_, reportFileName := filepath.Split(reportFile)
	if len(reportFileName) > maxReportFilenameLength {
		return nil, errors.New("report filename too long")
	}

	for _, format := range formats {
		switch format {
		case HtmlFormat:
			reportFileName = reportFile + ".html"
			err = printFullReportToHtml(s, reportFileName, reportTime, wafName, url, openApiFile, args, ignoreUnresolved, includePayloads)
			if err != nil {
				return nil, err
			}

		case PdfFormat:
			reportFileName = reportFile + ".pdf"
			err = printFullReportToPdf(ctx, s, reportFileName, reportTime, wafName, url, openApiFile, args, ignoreUnresolved, includePayloads)
			if err != nil {
				return nil, err
			}

		case JsonFormat:
			reportFileName = reportFile + ".json"
			err = printFullReportToJson(s, reportFileName, reportTime, wafName, url, args, ignoreUnresolved)
			if err != nil {
				return nil, err
			}

		case NoneFormat:
			return nil, nil

		default:
			return nil, fmt.Errorf("unknown report format: %s", format)
		}

		reportFileNames = append(reportFileNames, reportFileName)
	}

	return reportFileNames, nil
}

func ValidateReportFormat(formats []string) error {
	if len(formats) == 0 {
		return errors.New("no report format specified")
	}

	// Convert slice to set (map)
	set := make(map[string]any)
	for _, s := range formats {
		if _, ok := ReportFormatsSet[s]; !ok {
			return fmt.Errorf("unknown report format: %s", s)
		}

		set[s] = nil
	}

	// Check for duplicating values
	if len(set) != len(formats) {
		return fmt.Errorf("found duplicated values: %s", strings.Join(formats, ","))
	}

	// Check "none" is present
	_, isNone := set[NoneFormat]

	// Check for conflicts
	if len(set) > 1 && isNone {
		// Delete "none" from the set
		delete(set, NoneFormat)
		// Collect conflicted formats
		conflictedFormats := slices.Collect(maps.Keys(set))

		return fmt.Errorf("\"none\" conflicts with other formats: %s", strings.Join(conflictedFormats, ","))
	}

	return nil
}

func IsNoneReportFormat(reportFormat []string) bool {
	if len(reportFormat) > 0 && reportFormat[0] == NoneFormat {
		return true
	}

	return false
}

func IsPdfOrHtmlReportFormat(reportFormats []string) bool {
	for _, format := range reportFormats {
		if format == PdfFormat {
			return true
		}
		if format == HtmlFormat {
			return true
		}
	}

	return false
}
