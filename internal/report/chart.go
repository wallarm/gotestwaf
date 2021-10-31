package report

import (
	"bytes"
	"fmt"

	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

type chartPart struct {
	value int
	name  string
	color drawing.Color
}

func drawChart(parts []chartPart, overall int, title string) (*bytes.Buffer, error) {
	lastPartIndex := len(parts) - 1
	percents := make([]float64, len(parts))
	percents[lastPartIndex] = 100.0

	pieChartImg := chart.PieChart{
		DPI:   85,
		Title: title,
		TitleStyle: chart.Style{
			Show:              true,
			TextVerticalAlign: chart.TextVerticalAlignBaseline,
		},
		Background: chart.Style{
			Show:    true,
			Padding: chart.NewBox(25, 25, 25, 25),
		},
		Width:  512,
		Height: 512,
	}

	for i, part := range parts[:lastPartIndex] {
		percents[i] = float64(part.value*100) / float64(overall)
		percents[lastPartIndex] -= percents[i]
	}

	for i, part := range parts {
		pieChartImg.Values = append(pieChartImg.Values, chart.Value{
			Value: float64(part.value),
			Label: fmt.Sprintf("%s: %d (%.2f%%)", part.name, part.value, percents[i]),
			Style: chart.Style{
				FillColor: part.color,
				FontSize:  12,
			},
		})
	}

	buffer := bytes.NewBuffer([]byte{})
	if err := pieChartImg.Render(chart.PNG, buffer); err != nil {
		return buffer, err
	}

	return buffer, nil
}

func drawDetectionScoreChart(bypassed, blocked, failed, overall int) (*bytes.Buffer, error) {
	parts := []chartPart{
		{bypassed, "Bypassed", drawing.ColorFromAlphaMixedRGBA(234, 67, 54, 255)},
		{blocked, "Blocked", drawing.ColorFromAlphaMixedRGBA(66, 133, 244, 255)},
		{failed, "Failed", drawing.ColorFromAlphaMixedRGBA(193, 193, 193, 255)},
	}
	return drawChart(parts, overall, "Detection Score")
}

func drawPositiveTestScoreChart(bypassed, blocked, failed, overall int) (*bytes.Buffer, error) {
	parts := []chartPart{
		{bypassed, "Bypassed", drawing.ColorFromAlphaMixedRGBA(234, 67, 54, 255)},
		{blocked, "Blocked", drawing.ColorFromAlphaMixedRGBA(66, 133, 244, 255)},
		{failed, "Failed", drawing.ColorFromAlphaMixedRGBA(193, 193, 193, 255)},
	}
	return drawChart(parts, overall, "Positive Tests Score")
}
