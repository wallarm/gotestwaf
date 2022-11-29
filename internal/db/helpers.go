package db

import (
	"math"
	"strings"
)

type Integer interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64
}

type Float interface {
	float32 | float64
}

type Number interface {
	Integer | Float
}

func Round(n float64) float64 {
	if math.IsNaN(n) {
		return 0.0
	}

	return math.Round(n*100) / 100
}

func CalculatePercentage[A Number, B Number](first A, second B) float64 {
	if second == 0 {
		return 0.0
	}

	result := float64(first) / float64(second) * 100

	if math.IsNaN(result) {
		return 0.0
	}

	return Round(result)
}

func isPositiveTest(setName string) bool {
	return strings.Contains(setName, "false")
}

func isApiTest(setName string) bool {
	return strings.Contains(setName, "api")
}
