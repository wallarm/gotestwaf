package report

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"

	"github.com/wallarm/gotestwaf/internal/db"
)

var emptyIndicator = opts.Indicator{Name: "", Max: 100}

type pair struct {
	blocked  int
	bypassed int
}

func calculatePercentage(first, second int) float32 {
	if second == 0 {
		return 0.0
	}
	return float32(first) / float32(second) * 100
}

func updateCounters(t db.TestDetails, counters map[string]map[string]pair, isBlocked bool) {
	var category string
	var typ string

	if isApiTest(t.TestSet) {
		category = "api"
	} else {
		category = "app"
	}

	if t.Type == "" {
		typ = "unknown"
	} else {
		typ = strings.ToLower(t.Type)
	}

	if _, ok := counters[category]; !ok {
		counters[category] = make(map[string]pair)
	}

	val := counters[category][typ]
	if isBlocked {
		val.blocked++
	} else {
		val.bypassed++
	}
	counters[category][typ] = val
}

func getIndicatorsAndItems(counters map[string]map[string]pair, category string) (
	indicators []*opts.Indicator, items []opts.RadarData,
) {
	var values []float32

	for testType, val := range counters[category] {
		percentage := calculatePercentage(val.blocked, val.blocked+val.bypassed)
		indicators = append(indicators, &opts.Indicator{
			Name: fmt.Sprintf("%s (%.1f%%)", testType, percentage),
			Max:  100,
		})
		values = append(values, percentage)
	}

	if len(indicators) == 1 {
		indicators = append(indicators, &emptyIndicator, &emptyIndicator)
		values = append(values, 100, 100)
	} else if len(indicators) == 2 {
		indicators = append([]*opts.Indicator{&emptyIndicator}, indicators...)
		values = append([]float32{0}, values...)
	}

	if indicators != nil {
		items = []opts.RadarData{
			{Value: values},
		}
	}

	return
}

func generateCharts(s *db.Statistics) (apiChart *string, appChart *string, err error) {
	counters := make(map[string]map[string]pair)

	for _, t := range s.Blocked {
		updateCounters(t, counters, true)
	}

	for _, t := range s.Bypasses {
		updateCounters(t, counters, false)
	}

	apiIndicators, apiItems := getIndicatorsAndItems(counters, "api")
	appIndicators, appItems := getIndicatorsAndItems(counters, "app")

	var buffer bytes.Buffer
	re := regexp.MustCompile(`<script type="text/javascript">(\n|.)*</script>`)
	reRenderer := regexp.MustCompile(`(echarts\.init\()(.*)(\))`)

	if apiIndicators != nil {
		chart := charts.NewRadar()
		chart.SetGlobalOptions(
			charts.WithTitleOpts(opts.Title{
				Title: "API Security",
				Right: "center",
			}),
			charts.WithInitializationOpts(opts.Initialization{
				ChartID: "api_chart",
			}),
			charts.WithRadarComponentOpts(opts.RadarComponent{
				Indicator: apiIndicators,
				SplitArea: &opts.SplitArea{Show: true},
				SplitLine: &opts.SplitLine{Show: true},
			}),
		)
		chart.AddSeries("", apiItems)

		err = chart.Render(&buffer)
		if err != nil {
			return nil, nil, err
		}

		scriptParts := re.FindAllString(buffer.String(), -1)
		if len(scriptParts) != 1 {
			return nil, nil, errors.New("couldn't get chart script")
		}

		script := reRenderer.ReplaceAllString(scriptParts[0], "$1$2, {renderer: \"svg\"}$3")

		apiChart = &script

		buffer.Reset()
	}

	if appIndicators != nil {
		chart := charts.NewRadar()
		chart.SetGlobalOptions(
			charts.WithTitleOpts(opts.Title{
				Title: "Application Security",
				Right: "center",
			}),
			charts.WithInitializationOpts(opts.Initialization{
				ChartID: "app_chart",
			}),
			charts.WithRadarComponentOpts(opts.RadarComponent{
				Indicator: appIndicators,
				SplitArea: &opts.SplitArea{Show: true},
				SplitLine: &opts.SplitLine{Show: true},
			}),
		)
		chart.AddSeries("", appItems)

		err = chart.Render(&buffer)
		if err != nil {
			return nil, nil, err
		}

		scriptParts := re.FindAllString(buffer.String(), -1)
		if len(scriptParts) != 1 {
			return nil, nil, errors.New("couldn't get chart script")
		}

		script := reRenderer.ReplaceAllString(scriptParts[0], "$1$2, {renderer: \"svg\"}$3")

		appChart = &script
	}

	return
}
