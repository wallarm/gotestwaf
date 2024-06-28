package placeholder

import (
	"testing"

	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

func TestURLPath(t *testing.T) {
	tests := []struct {
		requestURL string
		payload    string
		reqURL     string
	}{
		{"http://example.com", "hello-world", "http://example.com/hello-world"},
		{"http://example.com/", "hello-world", "http://example.com/hello-world"},
		{"http://example.com////", "hello-world", "http://example.com/hello-world"},
		{"http://example.com", "/hello-world", "http://example.com//hello-world"},
		{"http://example.com/", "/hello-world", "http://example.com//hello-world"},
		{"http://example.com", "%0d%0aSet-Cookie:crlf=injection", "http://example.com/%0d%0aSet-Cookie:crlf=injection"},
		{"http://example.com/", "%0d%0aSet-Cookie:crlf=injection", "http://example.com/%0d%0aSet-Cookie:crlf=injection"},
		{"http://example.com", "//%0d%0aSet-Cookie:crlf=injection", "http://example.com///%0d%0aSet-Cookie:crlf=injection"},
		{"http://example.com/", "//%0d%0aSet-Cookie:crlf=injection", "http://example.com///%0d%0aSet-Cookie:crlf=injection"},
		{"http://example.com", "//%2f/a/b", "http://example.com///%2f/a/b"},
		{"http://example.com/", "//%2f/a/b", "http://example.com///%2f/a/b"},
	}

	for _, test := range tests {
		req, err := DefaultURLPath.CreateRequest(test.requestURL, test.payload, nil, types.GoHTTPClient)
		if err != nil {
			t.Fatalf("got an error while testing: %v", err)
		}

		r, ok := req.(*types.GoHTTPRequest)
		if !ok {
			t.Fatalf("bad request type: %T, expected %T", req, &types.GoHTTPRequest{})
		}

		if reqURL := r.Req.URL.String(); reqURL != test.reqURL {
			t.Fatalf("got %s, want %s", reqURL, test.reqURL)
		}
	}
}
