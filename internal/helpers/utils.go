package helpers

// DeepCopyMap is a generic function to copy a map with any key and value types.
func DeepCopyMap[K comparable, V any](original map[K]V) map[K]V {
	// Create a new map to hold the copy
	mapCopy := make(map[K]V)

	// Iterate over the original map and copy each key-value pair to the new map
	for key, value := range original {
		mapCopy[key] = value
	}

	return mapCopy
}
