package openapi

import "math/rand"

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// genRandomInt generates a random integer within the given bounds.
func genRandomInt(min, max *float64, exclusiveMin, exclusiveMax bool) int {
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
func genRandomFloat(min, max *float64, exclusiveMin, exclusiveMax bool) float64 {
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
