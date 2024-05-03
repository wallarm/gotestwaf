package helpers

import "net/url"

// GetTargetURL returns *url.URL with empty path, query and fragments parts.
func GetTargetURL(reqURL *url.URL) *url.URL {
	targetURL := *reqURL

	targetURL.Path = ""
	targetURL.RawPath = ""
	targetURL.ForceQuery = false
	targetURL.RawQuery = ""
	targetURL.Fragment = ""
	targetURL.RawFragment = ""

	return &targetURL
}

// GetTargetURLStr returns *url.URL with empty path, query and fragments parts
// as a string.
func GetTargetURLStr(reqURL *url.URL) string {
	targetURL := GetTargetURL(reqURL)

	return targetURL.String()
}
