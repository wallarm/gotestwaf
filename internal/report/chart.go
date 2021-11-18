package report

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"

	"github.com/wallarm/gotestwaf/internal/db"
)

func generateRadarItems(radarData [][]float32) []opts.RadarData {
	items := make([]opts.RadarData, 0)
	for i := 0; i < len(radarData); i++ {
		items = append(items, opts.RadarData{Name: "Name", Value: radarData[i]})
	}
	return items

}

func generateChartScript(summaryTable []db.SummaryTableRow) (*string, error) {
	// TODO: delete
	// var indicators []*opts.Indicator
	// var values []float32
	//
	// for _, row := range summaryTable {
	// 	indicators = append(indicators, &opts.Indicator{
	// 		Name: fmt.Sprintf("%s/%s (%.2f)", row.TestSet, row.TestCase, row.Percentage),
	// 		Max:  100,
	// 	})
	// 	values = append(values, row.Percentage)
	// }
	//
	// items := []opts.RadarData{
	// 	{Value: values},
	// }
	//
	// chart := charts.NewRadar()
	// chart.SetGlobalOptions(
	// 	charts.WithInitializationOpts(opts.Initialization{
	// 		ChartID: "chart",
	// 	}),
	// 	charts.WithRadarComponentOpts(opts.RadarComponent{
	// 		Indicator: indicators,
	// 		SplitArea: &opts.SplitArea{Show: true},
	// 		SplitLine: &opts.SplitLine{Show: true},
	// 	}),
	// )
	// chart.AddSeries("", items)

	var items []opts.PieData
	for _, row := range summaryTable {
		items = append(items, opts.PieData{
			Name:  fmt.Sprintf("%s/%s: %.2f%%", row.TestSet, row.TestCase, row.Percentage),
			Value: row.Percentage,
		})
	}

	chart := charts.NewPie()
	chart.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			ChartID: "chart",
		}),
	)

	chart.AddSeries("chart", items).
		SetSeriesOptions(
			charts.WithLabelOpts(opts.Label{
				Show:      true,
				Formatter: "{b}",
			}),
			charts.WithPieChartOpts(opts.PieChart{
				Radius:   []string{"20%", "50%"},
				RoseType: "area",
			}),
		)

	var buffer bytes.Buffer

	err := chart.Render(&buffer)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`<script type="text/javascript">(\n|.)*</script>`)
	scriptParts := re.FindAllString(buffer.String(), -1)
	if len(scriptParts) != 1 {
		return nil, errors.New("couldn't get chart script")
	}

	re = regexp.MustCompile(`let`)
	script := re.ReplaceAllString(scriptParts[0], "var")

	// TODO: delete
	// re = regexp.MustCompile(`"white"`)
	// script = re.ReplaceAllString(script, `"white", {renderer: "svg"}`)

	return &script, nil
}
