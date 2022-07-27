package detectors

import (
	"net/http"
	"regexp"
)

// Check performs some check on the response with a fixed condition.
type Check func(resp *http.Response) bool

// CheckStatusCode compare response status code with given value.
func CheckStatusCode(status int) Check {
	f := func(resp *http.Response) bool {
		if resp.StatusCode == status {
			return true
		}

		return false
	}

	return f
}

// CheckHeader match header value with regex.
func CheckHeader(header, regex string) Check {
	re := regexp.MustCompile(regex)

	f := func(resp *http.Response) bool {
		values := resp.Header.Values(header)
		if values == nil {
			return false
		}

		for i := range values {
			if re.MatchString(values[i]) {
				return true
			}
		}

		return false
	}

	return f
}

// CheckCookie match Set-Cookie header values with regex.
func CheckCookie(regex string) Check {
	return CheckHeader("Set-Cookie", regex)
}

// CheckContent match body value with regex.
func CheckContent(regex string) Check {
	re := regexp.MustCompile(regex)

	f := func(resp *http.Response) bool {
		var body []byte

		_, err := resp.Body.Read(body)
		if err != nil {
			return false
		}

		if re.Match(body) {
			return true
		}

		return false
	}

	return f
}
