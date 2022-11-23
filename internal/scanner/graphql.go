package scanner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

// List of the possible GraphQL endpoints on URL.
var checkAvailabilityEndpoints = []string{
	"/graphql",
	"/_graphql",
	"/api/graphql",
	"/GraphQL",
}

// checkAnswer checks that answer contains "__typename" in the response body.
// Example of correct answer:
//
//	{
//	  "data": {
//	    "__typename": "Query"
//	  }
//	}
func checkAnswer(body []byte) (bool, error) {
	jsonMap := make(map[string]any)

	err := json.Unmarshal(body, &jsonMap)
	if err != nil {
		return false, errors.Wrap(err, "couldn't unmarshal JSON")
	}

	data, ok := jsonMap["data"]
	if !ok {
		return false, nil
	}

	dataMap, ok := data.(map[string]any)
	if !ok {
		return false, nil
	}

	_, ok = dataMap["__typename"]
	if ok {
		return true, nil
	}

	return false, nil
}

func (s *Scanner) checkGraphQlAvailability(ctx context.Context) (bool, error) {
	endpointsToCheck := checkAvailabilityEndpoints

	s.httpClient.isGraphQlAvailable = false

	endpointURL, _ := url.Parse(s.cfg.GraphQlURL)

	// Add query parameter to trigger GraphQL
	queryParams := endpointURL.Query()
	queryParams.Set("query", "{__typename}")
	endpointURL.RawQuery = queryParams.Encode()

	// If s.cfg.GraphQlURL is different from s.cfg.URL, we only need to check
	// one endpoint - s.cfg.GraphQlURL
	if s.cfg.GraphQlURL != s.cfg.URL {
		endpointsToCheck = []string{endpointURL.Path}
		endpointURL.Path = ""
	}

	for _, endpoint := range endpointsToCheck {
		endpointURL.Path = endpoint

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL.String(), nil)
		if err != nil {
			return false, errors.New("couldn't create request to check GraphQL availability")
		}

		_, body, statusCode, err := s.httpClient.SendRequest(req, "")
		if err != nil {
			return false, errors.New("couldn't send request to check GraphQL availability")
		}

		if statusCode == http.StatusOK {
			ok, err := checkAnswer([]byte(body))
			if err != nil {
				return false, errors.Wrap(err, "couldn't check response")
			}

			// If we found correct GraphQL endpoint, save it
			if ok {
				endpointURL.RawQuery = ""

				s.cfg.GraphQlURL = endpointURL.String()
				s.httpClient.isGraphQlAvailable = true

				return true, nil
			}
		}
	}

	return false, nil
}
