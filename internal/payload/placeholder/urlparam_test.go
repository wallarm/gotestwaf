package placeholder

import (
	"regexp"
	"testing"
)

func TestURLParam(t *testing.T) {
	tests := []struct {
		requestURL   string
		payload      string
		reqURLregexp string
	}{
		{"http://example.com", "hello-world", `http://example\.com/\?[a-f0-9]{10}=hello-world`},
		{"http://example.com/", "hello-world", `http://example\.com/\?[a-f0-9]{10}=hello-world`},
		{"http://example.com////", "hello-world", `http://example\.com/\?[a-f0-9]{10}=hello-world`},
		{"http://example.com/?a=b", "hello-world", `http://example\.com/\?a=b&[a-f0-9]{10}=hello-world`},
		{"http://example.com/?a=b#abc", "hello-world", `http://example\.com/\?a=b&[a-f0-9]{10}=hello-world`},
		{"http://example.com/a/b/c", "hello-world", `http://example\.com/a/b/c\?[a-f0-9]{10}=hello-world`},
		{"http://example.com/a/b/c/", "hello-world", `http://example\.com/a/b/c\?[a-f0-9]{10}=hello-world`},
		{"http://example.com/a/b/c?a=b", "hello-world", `http://example\.com/a/b/c\?a=b&[a-f0-9]{10}=hello-world`},
		{"http://example.com/a/b/c?a=b#abc", "hello-world", `http://example\.com/a/b/c\?a=b&[a-f0-9]{10}=hello-world`},
		{"http://example.com", "%0D%0A%09%2Fhello%25world%23", `http://example\.com/\?[a-f0-9]{10}=%0D%0A%09%2Fhello%25world%23`},
		{"http://example.com/", "%0D%0A%09%2Fhello%25world%23", `http://example\.com/\?[a-f0-9]{10}=%0D%0A%09%2Fhello%25world%23`},
		{"http://example.com////", "%0D%0A%09%2Fhello%25world%23", `http://example\.com/\?[a-f0-9]{10}=%0D%0A%09%2Fhello%25world%23`},
		{"http://example.com/?a=b", "%0D%0A%09%2Fhello%25world%23", `http://example\.com/\?a=b&[a-f0-9]{10}=%0D%0A%09%2Fhello%25world%23`},
		{"http://example.com/?a=b#abc", "%0D%0A%09%2Fhello%25world%23", `http://example\.com/\?a=b&[a-f0-9]{10}=%0D%0A%09%2Fhello%25world%23`},
		{"http://example.com/a/b/c", "%0D%0A%09%2Fhello%25world%23", `http://example\.com/a/b/c\?[a-f0-9]{10}=%0D%0A%09%2Fhello%25world%23`},
		{"http://example.com/a/b/c/", "%0D%0A%09%2Fhello%25world%23", `http://example\.com/a/b/c\?[a-f0-9]{10}=%0D%0A%09%2Fhello%25world%23`},
		{"http://example.com/a/b/c?a=b", "%0D%0A%09%2Fhello%25world%23", `http://example\.com/a/b/c\?a=b&[a-f0-9]{10}=%0D%0A%09%2Fhello%25world%23`},
		{"http://example.com/a/b/c?a=b#abc", "%0D%0A%09%2Fhello%25world%23", `http://example\.com/a/b/c\?a=b&[a-f0-9]{10}=%0D%0A%09%2Fhello%25world%23`},
	}

	for _, test := range tests {
		req, err := DefaultURLParam.CreateRequest(test.requestURL, test.payload)
		if err != nil {
			t.Fatalf("got an error while testing: %v", err)
		}

		reqURL := req.URL.String()
		matched, err := regexp.MatchString(test.reqURLregexp, reqURL)
		if err != nil {
			t.Fatalf("got an error while testing: %v", err)
		}
		if !matched {
			t.Fatalf("got %s, want %s", reqURL, test.reqURLregexp)
		}
	}
}
