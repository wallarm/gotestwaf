package db

import (
	"math"
	"strings"
)

func Round(n float64) float64 {
	return math.Round(n*100) / 100
}

func CalculatePercentage[A int | float32 | float64](first A, second int) float64 {
	if second == 0 {
		return 0.0
	}
	result := float64(first) / float64(second) * 100
	return Round(result)
}

func isPositiveTest(setName string) bool {
	return strings.Contains(setName, "false")
}

func isApiTest(setName string) bool {
	return strings.Contains(setName, "api")
}
