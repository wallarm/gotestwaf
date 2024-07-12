package types

import "fmt"

var _ fmt.Stringer = (*HTTPClientType)(nil)
var _ error = (*UnknownHTTPClientError)(nil)

// HTTPClientType represents the type of HTTP client.
type HTTPClientType int

// Enumeration of available HTTPClientType values.
const (
	// GoHTTPClient indicates that the standard Go HTTP client is used.
	GoHTTPClient HTTPClientType = iota

	// ChromeHTTPClient indicates that a Chrome-based HTTP client is used.
	ChromeHTTPClient
)

// String returns string representation of HTTPClientType type.
func (t HTTPClientType) String() string {
	switch t {
	case GoHTTPClient:
		return "GoHTTPClient"
	case ChromeHTTPClient:
		return "ChromeHTTPClient"
	default:
		return fmt.Sprintf("%d", int(t))
	}
}

// UnknownHTTPClientError represents an error for unknown HTTP client types.
type UnknownHTTPClientError struct {
	clientType HTTPClientType
}

func NewUnknownHTTPClientError(clientType HTTPClientType) *UnknownHTTPClientError {
	return &UnknownHTTPClientError{clientType: clientType}
}

func (e *UnknownHTTPClientError) Error() string {
	return fmt.Sprintf(
		"unknown HTTP client type: %s, expected %s or %s",
		e.clientType, GoHTTPClient, ChromeHTTPClient,
	)
}
