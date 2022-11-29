package report

import (
	"bytes"
	"regexp"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/pkg/errors"
)

const (
	titleColor = "#000000"
	labelColor = "#333333"

	maxValue       = 100.0
	emptyIndicator = "-"
	emptyItem      = maxValue
)

// GenerateCharts generates JS code to render charts in HTML/PDF report.
func generateCharts(
	apiIndicators []string, apiItems []float64,
	appIndicators []string, appItems []float64,
) (apiChart *string, appChart *string, err error) {
	var buffer bytes.Buffer

	re := regexp.MustCompile(`<script type="text/javascript">(\n|.)*</script>`)
	reRenderer := regexp.MustCompile(`(echarts\.init\()(.*)(\))`)

	if len(apiIndicators) != len(apiItems) ||
		len(appIndicators) != len(appItems) {
		return nil, nil, errors.New("the number of indicators does not match the number of values")
	}

	if apiIndicators != nil && len(apiIndicators) > 0 {
		var indicators []*opts.Indicator
		var items []opts.RadarData

		for i := 0; i < len(apiIndicators); i++ {
			indicators = append(indicators, &opts.Indicator{
				Name:  apiIndicators[i],
				Max:   maxValue,
				Color: labelColor,
			})

			if apiIndicators[i] == emptyIndicator {
				apiItems[i] = emptyItem
			}
		}

		items = []opts.RadarData{
			{Value: apiItems},
		}

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
				Indicator: indicators,
				SplitArea: &opts.SplitArea{Show: true},
				SplitLine: &opts.SplitLine{Show: true},
			}),
		)
		chart.AddSeries("", items)

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

	if appIndicators != nil && len(appIndicators) > 0 {
		var indicators []*opts.Indicator
		var items []opts.RadarData

		for i := 0; i < len(appIndicators); i++ {
			indicators = append(indicators, &opts.Indicator{
				Name:  appIndicators[i],
				Max:   maxValue,
				Color: labelColor,
			})

			if appIndicators[i] == emptyIndicator {
				appItems[i] = emptyItem
			}
		}

		items = []opts.RadarData{
			{Value: appItems},
		}

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
				Indicator: indicators,
				SplitArea: &opts.SplitArea{Show: true},
				SplitLine: &opts.SplitLine{Show: true},
			}),
		)
		chart.AddSeries("", items)

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
