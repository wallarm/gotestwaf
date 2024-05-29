package chrome

import (
	"fmt"
	"net/http"
)

// headersToMap converts network.Headers (map[string]interface{}) to
// http.Header (map[string][]string)
func headersToMap(headers map[string]interface{}) http.Header {
	result := make(http.Header)

	for k, v := range headers {
		switch v := v.(type) {
		case interface{}:
			result.Add(k, fmt.Sprintf("%v", v))
		case []interface{}:
			for _, val := range v {
				result.Add(k, fmt.Sprintf("%v", val))
			}
		default:
			result.Add(k, fmt.Sprintf("%v", v))
		}
	}

	return result
}
