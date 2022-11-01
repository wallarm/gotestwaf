package report

import "strings"

// MapKeysToString concatenates all keys of a map with a separator
// and returns a string.
func MapKeysToString(m map[string]interface{}, sep string) string {
	var keysList []string

	for k := range m {
		keysList = append(keysList, k)
	}

	return strings.Join(keysList, sep)
}
