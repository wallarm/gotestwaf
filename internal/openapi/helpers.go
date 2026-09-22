package openapi

import (
	"math/rand"

	"github.com/getkin/kin-openapi/openapi3"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// resolveBound converts an OpenAPI 3.0 boolean exclusive flag or an
// OpenAPI 3.1 numeric exclusive bound into a (bound, exclusive) pair.
func resolveBound(bound *float64, exclusive openapi3.ExclusiveBound) (*float64, bool) {
	if exclusive.Value != nil {
		return exclusive.Value, true
	}
	return bound, exclusive.IsTrue()
}

// genRandomInt generates a random integer within the given bounds.
func genRandomInt(min, max *float64, exclusiveMinBound, exclusiveMaxBound openapi3.ExclusiveBound) int {
	min, exclusiveMin := resolveBound(min, exclusiveMinBound)
	max, exclusiveMax := resolveBound(max, exclusiveMaxBound)

	minValue := 0
	maxValue := defaultMaxInt

	if min != nil {
		minValue = int(*min)
	}
	if max != nil {
		maxValue = int(*max)
	}

	if exclusiveMin {
		minValue++
	}
	if !exclusiveMax {
		maxValue++
	}

	randInt := minValue + rand.Intn(maxValue-minValue)

	return randInt
}

// genRandomFloat generates a random float within the given bounds.
func genRandomFloat(min, max *float64, exclusiveMinBound, exclusiveMaxBound openapi3.ExclusiveBound) float64 {
	min, exclusiveMin := resolveBound(min, exclusiveMinBound)
	max, exclusiveMax := resolveBound(max, exclusiveMaxBound)

	minValue := float64(0)
	maxValue := float64(defaultMaxInt)

	if min != nil {
		minValue = *min
	}
	if max != nil {
		maxValue = *max
	}

	if exclusiveMin {
		minValue++
	}
	if !exclusiveMax {
		maxValue++
	}

	randFloat := minValue + float64(rand.Intn(int(maxValue-minValue)))

	return randFloat
}

// genRandomString generates a random string of the right size.
func genRandomString(minLength, maxLength uint64) string {
	if minLength < defaultStringSize {
		minLength = defaultStringSize
	}

	randLength := int(minLength) + rand.Intn(int(maxLength-minLength+1))

	b := make([]rune, randLength)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}

	return string(b)
}

// genRandomPlaceholder generates a random placeholder with fixed size.
func genRandomPlaceholder() string {
	return genRandomString(defaultPlaceholderSize, defaultPlaceholderSize)
}
