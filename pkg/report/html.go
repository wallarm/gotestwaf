package report

import (
	"bytes"
	_ "embed"
	"html/template"
	"io"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/db"
)

//go:embed report_template.html
var HtmlTemplate string

// HtmlReport represents a data required to render a full report in HTML/PDF format.
type HtmlReport struct {
	IgnoreUnresolved bool `json:"ignore_unresolved" validate:"boolean"`

	WafName        string `json:"waf_name" validate:"required,alphanum"`
	Url            string `json:"url" validate:"required,url"`
	WafTestingDate string `json:"waf_testing_date" validate:"required,datetime=02 January 2006"`
	GtwVersion     string `json:"gtw_version" validate:"required,printascii"`
	TestCasesFP    string `json:"test_cases_fp" validate:"required,hexadecimal"`
	OpenApiFile    string `json:"open_api_file" validate:"omitempty,file"`
	Args           string `json:"args" validate:"required,printascii"`

	ApiSecChartData struct {
		Indicators []string       `json:"indicators" validate:"omitempty,dive,printascii"`
		Items      []float32      `json:"items" validate:"omitempty,dive,min=0,max=100"`
		Chart      *template.HTML `json:"-" validate:"-"`
	} `json:"api_sec_chart_data"`

	AppSecChartData struct {
		Indicators []string       `json:"indicators" validate:"omitempty,dive,printascii"`
		Items      []float32      `json:"items" validate:"omitempty,dive,min=0,max=100"`
		Chart      *template.HTML `json:"-" validate:"-"`
	} `json:"app_sec_chart_data"`

	Overall *Grade `json:"overall" validate:"required"`

	ApiSec struct {
		TrueNegative *Grade `json:"true_negative" validate:"required"`
		TruePositive *Grade `json:"true_positive" validate:"required"`
		Grade        *Grade `json:"grade" validate:"required"`
	} `json:"api_sec"`

	AppSec struct {
		TrueNegative *Grade `json:"true_negative" validate:"required"`
		TruePositive *Grade `json:"true_positive" validate:"required"`
		Grade        *Grade `json:"grade" validate:"required"`
	} `json:"app_sec"`

	ComparisonTable []*ComparisonTableRow `json:"comparison_table" validate:"required,dive,required"`

	TotalSent                int `json:"total_sent" validate:"min=0"`
	BlockedRequestsNumber    int `json:"blocked_requests_number" validate:"min=0"`
	BypassedRequestsNumber   int `json:"bypassed_requests_number" validate:"min=0"`
	UnresolvedRequestsNumber int `json:"unresolved_requests_number" validate:"min=0"`
	FailedRequestsNumber     int `json:"failed_requests_number" validate:"min=0"`

	ScannedPaths db.ScannedPaths `json:"scanned_paths" validate:"omitempty,dive,required"`

	NegativeTests struct {
		SummaryTable map[string]*TestSetSummary `json:"summary_table" validate:"omitempty,dive,keys,required,endkeys,required"`

		// map[paths]map[payload]map[statusCode]*testDetails
		Bypassed map[string]map[string]map[int]*TestDetails `json:"bypassed" validate:"omitempty,dive,keys,omitempty,endkeys,required,dive,keys,required,endkeys,required,dive,keys,min=0,endkeys,required"`
		// map[payload]map[statusCode]*testDetails
		Unresolved map[string]map[int]*TestDetails `json:"unresolved" validate:"omitempty,dive,keys,required,endkeys,required,dive,keys,min=0,endkeys,required"`
		Failed     []*db.FailedDetails             `json:"failed" validate:"omitempty,dive,required"`

		Percentage               float64 `json:"percentage" validate:"min=0,max=100"`
		TotalSent                int     `json:"total_sent" validate:"min=0"`
		BlockedRequestsNumber    int     `json:"blocked_requests_number" validate:"min=0"`
		BypassedRequestsNumber   int     `json:"bypassed_requests_number" validate:"min=0"`
		UnresolvedRequestsNumber int     `json:"unresolved_requests_number" validate:"min=0"`
		FailedRequestsNumber     int     `json:"failed_requests_number" validate:"min=0"`
	} `json:"negative_tests"`

	PositiveTests struct {
		SummaryTable map[string]*TestSetSummary `json:"summary_table" validate:"omitempty,dive,keys,required,endkeys,required"`

		// map[payload]map[statusCode]*testDetails
		Blocked map[string]map[int]*TestDetails `json:"blocked" validate:"omitempty,dive,keys,required,endkeys,required,dive,keys,min=0,endkeys,required"`
		// map[payload]map[statusCode]*testDetails
		Bypassed map[string]map[int]*TestDetails `json:"bypassed" validate:"omitempty,dive,keys,required,endkeys,required,dive,keys,min=0,endkeys,required"`
		// map[payload]map[statusCode]*testDetails
		Unresolved map[string]map[int]*TestDetails `json:"unresolved" validate:"omitempty,dive,keys,required,endkeys,required,dive,keys,min=0,endkeys,required"`
		Failed     []*db.FailedDetails             `json:"failed" validate:"omitempty,dive,required"`

		Percentage               float64 `json:"percentage" validate:"min=0,max=100"`
		TotalSent                int     `json:"total_sent" validate:"min=0"`
		BlockedRequestsNumber    int     `json:"blocked_requests_number" validate:"min=0"`
		BypassedRequestsNumber   int     `json:"bypassed_requests_number" validate:"min=0"`
		UnresolvedRequestsNumber int     `json:"unresolved_requests_number" validate:"min=0"`
		FailedRequestsNumber     int     `json:"failed_requests_number" validate:"min=0"`
	} `json:"positive_tests"`
}

type Grade struct {
	Percentage     float32 `json:"percentage" validate:"min=0,max=100"`
	Mark           string  `json:"mark" validate:"required,min=1,max=3"`
	CSSClassSuffix string  `json:"css_class_suffix" validate:"required,min=1,max=2"`
}

type ComparisonTableRow struct {
	Name         string `json:"name" validate:"required,printascii"`
	ApiSec       *Grade `json:"api_sec" validate:"required"`
	AppSec       *Grade `json:"app_sec" validate:"required"`
	OverallScore *Grade `json:"overall_score" validate:"required"`
}

type TestDetails struct {
	TestCase     string         `json:"test_case" validate:"required,printascii"`
	Encoders     map[string]any `json:"encoders" validate:"required,dive,keys,required,endkeys,omitempty"`
	Placeholders map[string]any `json:"placeholders" validate:"required,dive,keys,required,endkeys,omitempty"`
}

type TestSetSummary struct {
	TestCases []*db.SummaryTableRow `json:"test_cases" validate:"required,dive,required"`

	Percentage float64 `json:"percentage" validate:"min=0,max=100"`
	Sent       int     `json:"sent" validate:"min=0"`
	Blocked    int     `json:"blocked" validate:"min=0"`
	Bypassed   int     `json:"bypassed" validate:"min=0"`
	Unresolved int     `json:"unresolved" validate:"min=0"`
	Failed     int     `json:"failed" validate:"min=0"`

	ResolvedTestCasesNumber int `json:"resolved_test_cases_number" validate:"min=0"`
}

// RenderFullReportToHTML substitutes report data into HTML template.
func RenderFullReportToHTML(reportData *HtmlReport) (*bytes.Buffer, error) {
	apiChart, appChart, err := generateCharts(
		reportData.ApiSecChartData.Indicators, reportData.ApiSecChartData.Items,
		reportData.AppSecChartData.Indicators, reportData.AppSecChartData.Items,
	)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't generate chart scripts")
	}

	if apiChart != nil {
		v := template.HTML(*apiChart)
		reportData.ApiSecChartData.Chart = &v
	}
	if appChart != nil {
		v := template.HTML(*appChart)
		reportData.AppSecChartData.Chart = &v
	}

	templ := template.Must(
		template.New("report").
			Funcs(template.FuncMap{
				"script": func(s string) template.HTML {
					return template.HTML(s)
				},
				"StringsJoin":     strings.Join,
				"StringsSplit":    strings.Split,
				"MapKeysToString": MapKeysToString,
			}).
			Parse(HtmlTemplate))

	var buffer bytes.Buffer

	err = templ.Execute(io.MultiWriter(&buffer), reportData)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't execute template")
	}

	return &buffer, nil
}

// ValidateReportData validates report data
func ValidateReportData(reportData *HtmlReport) error {
	validate := validator.New()
	err := validate.Struct(reportData)
	if err != nil {
		return errors.Wrap(err, "found invalid values in the report data")
	}

	return nil
}
