package report

import "strings"

// isApiTest checks if a set of tests has the API category.
func isApiTest(setName string) bool {
	return strings.Contains(setName, "api")
}
