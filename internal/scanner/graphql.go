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
	s.httpClient.isGraphQlAvailable = false

	endpointURL, _ := url.Parse(s.cfg.GraphQlURL)

	// Add query parameter to trigger GraphQL
	queryParams := endpointURL.Query()
	queryParams.Set("query", "{__typename}")
	endpointURL.RawQuery = queryParams.Encode()

	// checker
	check := func(url *url.URL) (bool, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
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

			if ok {
				return true, nil
			}
		}

		return false, nil
	}

	// If GraphQlURL is the same as URL, search for the correct endpoint
	if s.cfg.GraphQlURL == s.cfg.URL {
		for _, endpoint := range checkAvailabilityEndpoints {
			endpointURL.Path = endpoint

			ok, err := check(endpointURL)
			if err != nil {
				return false, err
			}

			// If we found correct GraphQL endpoint, save it
			if ok {
				endpointURL.RawQuery = ""

				s.cfg.GraphQlURL = endpointURL.String()
				s.httpClient.isGraphQlAvailable = true

				return true, nil
			}
		}

		return false, nil
	}

	ok, err := check(endpointURL)
	if err != nil {
		return false, err
	}

	// If we found correct GraphQL endpoint, save it
	if ok {
		endpointURL.RawQuery = ""

		s.cfg.GraphQlURL = endpointURL.String()
		s.httpClient.isGraphQlAvailable = true

		return true, nil
	}

	return false, nil
}
