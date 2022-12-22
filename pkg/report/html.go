package report

import (
	"bytes"
	_ "embed"
	"html/template"
	"io"
	"strings"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/db"
)

//go:embed report_template.html
var HtmlTemplate string

// HtmlReport represents a data required to render a full report in HTML/PDF format.
type HtmlReport struct {
	IgnoreUnresolved bool `json:"ignore_unresolved" validate:"boolean"`
	IncludePayloads  bool `json:"include_payloads" validate:"boolean"`

	WafName        string   `json:"waf_name" validate:"required,printascii,max=256"`
	Url            string   `json:"url" validate:"required,url,max=256"`
	WafTestingDate string   `json:"waf_testing_date" validate:"required,datetime=02 January 2006"`
	GtwVersion     string   `json:"gtw_version" validate:"required,gtw_version"`
	TestCasesFP    string   `json:"test_cases_fp" validate:"required,fp"`
	OpenApiFile    string   `json:"open_api_file" validate:"omitempty,printascii,max=512"`
	Args           []string `json:"args" validate:"required,max=50,dive,args,max=200"`

	ApiSecChartData struct {
		Indicators []string       `json:"indicators" validate:"omitempty,max=100,dive,indicator"`
		Items      []float64      `json:"items" validate:"omitempty,max=100,dive,min=0,max=100"`
		Chart      *template.HTML `json:"-" validate:"-"`
	} `json:"api_sec_chart_data"`

	AppSecChartData struct {
		Indicators []string       `json:"indicators" validate:"omitempty,max=100,dive,indicator"`
		Items      []float64      `json:"items" validate:"omitempty,max=100,dive,min=0,max=100"`
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

	ScannedPaths db.ScannedPaths `json:"scanned_paths" validate:"omitempty,max=2048,dive,required"`

	NegativeTests struct {
		SummaryTable map[string]*TestSetSummary `json:"summary_table" validate:"omitempty,dive,keys,required,max=256,endkeys,required"`

		// map[paths]map[payload]map[statusCode]*testDetails
		Bypassed map[string]map[string]map[int]*TestDetails `json:"bypassed" validate:"omitempty,dive,keys,omitempty,endkeys,required,dive,keys,required,max=256000,endkeys,required,dive,keys,min=0,endkeys,required"`
		// map[payload]map[statusCode]*testDetails
		Unresolved map[string]map[int]*TestDetails `json:"unresolved" validate:"omitempty,dive,keys,required,max=256000,endkeys,required,dive,keys,min=0,endkeys,required"`
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
		Blocked map[string]map[int]*TestDetails `json:"blocked" validate:"omitempty,dive,keys,required,max=256000,endkeys,required,dive,keys,min=0,endkeys,required"`
		// map[payload]map[statusCode]*testDetails
		Bypassed map[string]map[int]*TestDetails `json:"bypassed" validate:"omitempty,dive,keys,required,max=256000,endkeys,required,dive,keys,min=0,endkeys,required"`
		// map[payload]map[statusCode]*testDetails
		Unresolved map[string]map[int]*TestDetails `json:"unresolved" validate:"omitempty,dive,keys,required,max=256000,endkeys,required,dive,keys,min=0,endkeys,required"`
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
	Percentage     float64 `json:"percentage" validate:"min=0,max=100"`
	Mark           string  `json:"mark" validate:"required,mark"`
	CSSClassSuffix string  `json:"css_class_suffix" validate:"required,css_suffix"`
}

type ComparisonTableRow struct {
	Name         string `json:"name" validate:"required,printascii,max=256"`
	ApiSec       *Grade `json:"api_sec" validate:"required"`
	AppSec       *Grade `json:"app_sec" validate:"required"`
	OverallScore *Grade `json:"overall_score" validate:"required"`
}

type TestDetails struct {
	TestCase     string         `json:"test_case" validate:"required,printascii,max=256"`
	Encoders     map[string]any `json:"encoders" validate:"required,encoders"`
	Placeholders map[string]any `json:"placeholders" validate:"required,placeholders"`
}

type TestSetSummary struct {
	TestCases []*db.SummaryTableRow `json:"test_cases" validate:"required,max=1024,dive,required"`

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
				"HTMLEscapeSlice": func(s []string) []string {
					escapedSlice := make([]string, len(s))
					for i := range s {
						escapedSlice[i] = template.HTMLEscapeString(s[i])
					}
					return escapedSlice
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
