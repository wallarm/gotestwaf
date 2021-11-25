package placeholder

import "testing"

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
		req, err := DefaultURLPath.CreateRequest(test.requestURL, test.payload)
		if err != nil {
			t.Fatalf("got an error while testing: %v", err)
		}

		if reqURL := req.URL.String(); reqURL != test.reqURL {
			t.Fatalf("got %s, want %s", reqURL, test.reqURL)
		}
	}
}
