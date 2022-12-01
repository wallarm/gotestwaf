package report

import (
	"fmt"
	"strings"

	"github.com/wallarm/gotestwaf/internal/db"
)

const (
	emptyIndicator = "-"
)

type pair struct {
	blocked  int
	bypassed int
}

// updateCounters counts tests by category.
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

// getIndicatorsAndItems returns indicators and values for charts.
func getIndicatorsAndItems(
	counters map[string]map[string]pair,
	category string,
) (indicators []string, items []float64) {
	for testType, val := range counters[category] {
		percentage := float64(db.CalculatePercentage(val.blocked, val.blocked+val.bypassed))

		indicators = append(indicators, fmt.Sprintf("%s (%.1f%%)", testType, percentage))
		items = append(items, percentage)
	}

	switch len(indicators) {
	case 0:
		return nil, nil

	case 1:
		indicators = []string{
			indicators[0], emptyIndicator, emptyIndicator,
			emptyIndicator, emptyIndicator, emptyIndicator,
		}
		items = []float64{
			items[0], 0.0, 0.0,
			0.0, 0.0, 0.0,
		}

	case 2:
		indicators = []string{
			emptyIndicator, indicators[0], emptyIndicator,
			emptyIndicator, indicators[1], emptyIndicator,
		}
		items = []float64{
			0.0, items[0], 0.0,
			0.0, items[1], 0.0,
		}

	case 3:
		indicators = []string{
			indicators[0], emptyIndicator, indicators[1],
			emptyIndicator, indicators[2], emptyIndicator,
		}
		items = []float64{
			items[0], 0.0, items[1],
			0.0, items[2], 0.0,
		}

	case 4:
		indicators = []string{
			emptyIndicator, indicators[0],
			emptyIndicator, indicators[1],
			emptyIndicator, indicators[2],
			emptyIndicator, indicators[3],
		}
		items = []float64{
			0.0, items[0],
			0.0, items[1],
			0.0, items[2],
			0.0, items[3],
		}
	}

	return
}

// generateChartData generates indicators and their values for JS charts.
func generateChartData(s *db.Statistics) (
	apiIndicators []string, apiItems []float64,
	appIndicators []string, appItems []float64,
) {
	counters := make(map[string]map[string]pair)

	for _, t := range s.NegativeTests.Blocked {
		updateCounters(t, counters, true)
	}

	for _, t := range s.NegativeTests.Bypasses {
		updateCounters(t, counters, false)
	}

	_, containsApiCat := counters["api"]

	// Add gRPC counter if gRPC is unavailable to display it on graphic
	if !s.IsGrpcAvailable && containsApiCat {
		// gRPC is part of the API Security tests
		counters["api"]["grpc"] = pair{}
	}

	apiIndicators, apiItems = getIndicatorsAndItems(counters, "api")
	appIndicators, appItems = getIndicatorsAndItems(counters, "app")

	// Fix label for gRPC if it is unavailable
	if !s.IsGrpcAvailable && containsApiCat {
		for i := 0; i < len(apiIndicators); i++ {
			if strings.HasPrefix(apiIndicators[i], "grpc") {
				apiIndicators[i] = "grpc (unavailable)"
				apiItems[i] = float64(0)
			}
		}
	}

	return
}
