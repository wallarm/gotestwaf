package helpers

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

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

// HostPortFromUrl returns is TLS required flag and host:port string.
func HostPortFromUrl(wafURL string, port uint16) (isTLS bool, hostPort string, err error) {
	urlParse, err := url.Parse(wafURL)
	if err != nil {
		return isTLS, "", err
	}

	host, _, err := net.SplitHostPort(urlParse.Host)
	if err != nil {
		if strings.Contains(err.Error(), "port") {
			host = urlParse.Host
		} else {
			return false, "", err
		}
	}

	host = net.JoinHostPort(host, fmt.Sprintf("%d", port))

	if urlParse.Scheme == "https" {
		isTLS = true
	}

	return isTLS, host, nil
}
