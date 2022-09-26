package report

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/db"
)

const (
	titleColor = "#000000"
	labelColor = "#333333"
)

var (
	emptyIndicator = opts.Indicator{Name: "-",
		Max:   100,
		Color: labelColor,
	}

	emptyValue = float32(100)
)

type pair struct {
	blocked  int
	bypassed int
}

func updateCounters(t *db.TestDetails, counters map[string]map[string]pair, isBlocked bool) {
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
		percentage := float32(db.CalculatePercentage(val.blocked, val.blocked+val.bypassed))
		indicators = append(indicators, &opts.Indicator{
			Name:  fmt.Sprintf("%s (%.1f%%)", testType, percentage),
			Max:   100,
			Color: labelColor,
		})
		values = append(values, percentage)
	}

	switch len(indicators) {
	case 0:
		return nil, nil

	case 1:
		indicators = []*opts.Indicator{
			indicators[0], &emptyIndicator, &emptyIndicator,
			&emptyIndicator, &emptyIndicator, &emptyIndicator,
		}
		values = []float32{
			values[0], emptyValue, emptyValue,
			emptyValue, emptyValue, emptyValue,
		}

	case 2:
		indicators = []*opts.Indicator{
			&emptyIndicator, indicators[0], &emptyIndicator,
			&emptyIndicator, indicators[1], &emptyIndicator,
		}
		values = []float32{
			emptyValue, values[0], emptyValue,
			emptyValue, values[1], emptyValue,
		}

	case 3:
		indicators = []*opts.Indicator{
			indicators[0], &emptyIndicator, indicators[1],
			&emptyIndicator, indicators[2], &emptyIndicator,
		}
		values = []float32{
			values[0], emptyValue, values[1],
			emptyValue, values[2], emptyValue,
		}

	case 4:
		indicators = []*opts.Indicator{
			&emptyIndicator, indicators[0],
			&emptyIndicator, indicators[1],
			&emptyIndicator, indicators[2],
			&emptyIndicator, indicators[3],
		}
		values = []float32{
			emptyValue, values[0],
			emptyValue, values[1],
			emptyValue, values[2],
			emptyValue, values[3],
		}
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

	for _, t := range s.NegativeTests.Blocked {
		updateCounters(t, counters, true)
	}

	for _, t := range s.NegativeTests.Bypasses {
		updateCounters(t, counters, false)
	}

	// Add gRPC counter if gRPC is unavailable to display it on graphic
	if !s.IsGrpcAvailable {
		// gRPC is part of the API Security tests
		counters["api"]["grpc"] = pair{}
	}

	apiIndicators, apiItems := getIndicatorsAndItems(counters, "api")
	appIndicators, appItems := getIndicatorsAndItems(counters, "app")

	// Fix label for gRPC if it is unavailable
	if !s.IsGrpcAvailable {
		for i := 0; i < len(apiIndicators); i++ {
			if strings.HasPrefix(apiIndicators[i].Name, "grpc") {
				apiIndicators[i].Name = "grpc (unavailable)"
				apiItems[0].Value.([]float32)[i] = float32(0)
			}
		}
	}

	var buffer bytes.Buffer
	re := regexp.MustCompile(`<script type="text/javascript">(\n|.)*</script>`)
	reRenderer := regexp.MustCompile(`(echarts\.init\()(.*)(\))`)

	if apiIndicators != nil {
		chart := charts.NewRadar()
		chart.SetGlobalOptions(
			charts.WithTitleOpts(opts.Title{
				Title: "API Security",
				Right: "center",
				TitleStyle: &opts.TextStyle{
					Color: titleColor,
				},
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
			return nil, nil, errors.Wrap(err, "couldn't render chart")
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
				TitleStyle: &opts.TextStyle{
					Color: titleColor,
				},
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
			return nil, nil, errors.Wrap(err, "couldn't render chart")
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
