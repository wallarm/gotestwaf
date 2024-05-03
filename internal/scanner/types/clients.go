package types

// HTTPClientType represents the type of HTTP client.
type HTTPClientType int

// Enumeration of available HTTPClientType values.
const (
	// GoHTTPClient indicates that the standard Go HTTP client is used.
	GoHTTPClient HTTPClientType = iota

	// ChromeHTTPClient indicates that a Chrome-based HTTP client is used.
	ChromeHTTPClient
)
