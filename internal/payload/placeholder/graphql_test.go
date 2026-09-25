package placeholder

import (
	"regexp"
	"testing"

	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

func TestGraphQLGet(t *testing.T) {
	tests := []struct {
		requestURL   string
		payload      string
		reqURLregexp string
	}{
		{"http://example.com/graphql", "hello-world", `^http://example\.com/graphql\?query=hello-world$`},
		{"http://example.com/graphql?a=b", "hello-world", `^http://example\.com/graphql\?a=b&query=hello-world$`},
		// Payload already encoded by the URL encoder must not be encoded again.
		{"http://example.com/graphql", "%7B__schema%7Btypes%7Bname%7D%7D%7D", `^http://example\.com/graphql\?query=%7B__schema%7Btypes%7Bname%7D%7D%7D$`},
		{"http://example.com/graphql?a=b", "%7B__schema%7Btypes%7Bname%7D%7D%7D", `^http://example\.com/graphql\?a=b&query=%7B__schema%7Btypes%7Bname%7D%7D%7D$`},
	}

	config, err := DefaultGraphQL.NewPlaceholderConfig(map[any]any{"method": "GET"})
	if err != nil {
		t.Fatalf("got an error while creating config: %v", err)
	}

	for _, test := range tests {
		req, err := DefaultGraphQL.CreateRequest(test.requestURL, test.payload, config, types.GoHTTPClient)
		if err != nil {
			t.Fatalf("got an error while testing: %v", err)
		}

		r, ok := req.(*types.GoHTTPRequest)
		if !ok {
			t.Fatalf("bad request type: %T, expected %T", req, &types.GoHTTPRequest{})
		}

		reqURL := r.Req.URL.String()
		matched, err := regexp.MatchString(test.reqURLregexp, reqURL)
		if err != nil {
			t.Fatalf("got an error while testing: %v", err)
		}
		if !matched {
			t.Fatalf("got %s, want %s", reqURL, test.reqURLregexp)
		}
	}
}
