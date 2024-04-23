package detectors

import (
	"bytes"
	"io"
	"net/http"
	"regexp"
	"strings"
)

type Responses struct {
	Resp         *http.Response
	RespToAttack *http.Response
}

// Check performs some check on the response with a fixed condition.
type Check func(resps *Responses) bool

// CheckStatusCode compare response status code with given value.
// Default value for attack parameter is true.
func CheckStatusCode(status int, attack bool) Check {
	f := func(resps *Responses) bool {
		resp := resps.Resp
		if attack {
			resp = resps.RespToAttack
		}

		if resp == nil {
			return false
		}

		if resp.StatusCode == status {
			return true
		}

		return false
	}

	return f
}

// CheckReason match status reason value with regex.
// Default value for attack parameter is true.
func CheckReason(regex string, attack bool) Check {
	re := regexp.MustCompile(regex)

	f := func(resps *Responses) bool {
		resp := resps.Resp
		if attack {
			resp = resps.RespToAttack
		}

		if resp == nil {
			return false
		}

		reasonIndex := strings.Index(resp.Status, " ")
		reason := resp.Status[reasonIndex+1:]

		if re.MatchString(reason) {
			return true
		}

		return false
	}

	return f
}

// CheckHeader match header value with regex.
// Default value for attack parameter is false.
func CheckHeader(header, regex string, attack bool) Check {
	re := regexp.MustCompile(regex)

	f := func(resps *Responses) bool {
		resp := resps.Resp
		if attack {
			resp = resps.RespToAttack
		}

		if resp == nil {
			return false
		}

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
// Default value for attack parameter is false.
func CheckCookie(regex string, attack bool) Check {
	return CheckHeader("Set-Cookie", regex, attack)
}

// CheckContent match body value with regex.
// Default value for attack parameter is true.
func CheckContent(regex string, attack bool) Check {
	re := regexp.MustCompile(regex)

	f := func(resps *Responses) bool {
		resp := resps.Resp
		if attack {
			resp = resps.RespToAttack
		}

		if resp == nil {
			return false
		}

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

// And combines the checks with AND logic,
// so each test must be true to return true.
func And(checks ...Check) Check {
	f := func(resps *Responses) bool {
		for _, check := range checks {
			if !check(resps) {
				return false
			}
		}

		return true
	}

	return f
}

// Or combines the checks with OR logic,
// so at least one test must be true to return true.
func Or(checks ...Check) Check {
	f := func(resps *Responses) bool {
		for _, check := range checks {
			if check(resps) {
				return true
			}
		}

		return false
	}

	return f
}
