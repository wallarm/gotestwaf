package detectors

import (
	"bytes"
	"io"
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
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false
		}

		// body reuse
		resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(body))

		if re.Match(body) {
			return true
		}

		return false
	}

	return f
}
