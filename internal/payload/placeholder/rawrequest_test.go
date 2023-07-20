package placeholder

import (
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"
)

func TestRawRequestNewConfig(t *testing.T) {
	type checkFunc func(c any, e error)

	var conf RawRequestConfig

	getRef := func(s string) *string {
		return &s
	}

	checkErr := func(c any, e error) {
		if c != nil {
			t.Error("config (*RawRequestConfig) must be nil")
		}

		var err *BadPlaceholderConfigError
		if !errors.As(e, &err) {
			t.Errorf("e should be an %T, got %T", err, e)
		}
	}

	checkValue := func(field any, value any) checkFunc {
		return func(c any, e error) {
			typedConf, ok := c.(*RawRequestConfig)
			if !ok {
				t.Errorf("bad conf type, got %v, expected %s", c, &conf)
			}

			if e != nil {
				t.Error("e (error) must be nil")
			}

			conf = *typedConf
			if !reflect.DeepEqual(field, value) {
				t.Errorf("bad value, got %v, expected %v", field, value)
			}
		}
	}

	tests := []struct {
		conf       map[any]any
		checkValue checkFunc
	}{
		{conf: map[any]any{}, checkValue: checkErr},
		{conf: map[any]any{"method": 0}, checkValue: checkErr},
		{conf: map[any]any{"method": "POST", "path": 0}, checkValue: checkErr},
		{conf: map[any]any{"method": "POST", "path": "/", "headers": 0}, checkValue: checkErr},
		{conf: map[any]any{"method": "POST", "path": "/", "headers": map[any]any{0: 0}}, checkValue: checkErr},
		{conf: map[any]any{"method": "POST", "path": "/", "headers": map[any]any{"X-Test": 0}}, checkValue: checkErr},
		{conf: map[any]any{"method": "POST", "path": "/"}, checkValue: checkValue(&conf.Method, getRef("POST"))},
		{conf: map[any]any{"method": "abcd", "path": "/"}, checkValue: checkValue(&conf.Method, getRef("abcd"))},
		{conf: map[any]any{"method": "POST"}, checkValue: checkValue(&conf.Path, getRef("/"))},
		{conf: map[any]any{"method": "POST", "path": "/abcd/{{payload}}"}, checkValue: checkValue(&conf.Path, getRef("/abcd/{{payload}}"))},
		{conf: map[any]any{"method": "POST", "headers": map[any]any{"X-Test": "Test Header {{payload}}"}}, checkValue: checkValue(&conf.Headers, &map[string]string{"X-Test": "Test Header {{payload}}"})},
		{conf: map[any]any{"method": "POST", "body": "Test {{payload}}"}, checkValue: checkValue(&conf.Body, getRef("Test {{payload}}"))},
	}

	for _, test := range tests {
		conf, err := DefaultRawRequest.newConfig(test.conf)
		test.checkValue(conf, err)
	}
}

func TestRawRequest(t *testing.T) {
	type checkFunc func(r *http.Request)

	const (
		url         = "http://example.com/"
		testPayload = "0123456789abcdef"
	)

	tests := []struct {
		conf       *RawRequestConfig
		checkValue checkFunc
	}{
		{
			conf: &RawRequestConfig{
				Method:  "POST",
				Path:    "/",
				Headers: make(map[string]string),
			},
			checkValue: func(r *http.Request) {
				got := r.Method
				expected := "POST"

				if !(got == expected) {
					t.Errorf("test failed, got %s, expected %s", got, expected)
				}
			},
		},
		{
			conf: &RawRequestConfig{
				Method:  "abcd",
				Path:    "/",
				Headers: make(map[string]string),
			},
			checkValue: func(r *http.Request) {
				got := r.Method
				expected := "abcd"

				if !(got == expected) {
					t.Errorf("test failed, got %s, expected %s", got, expected)
				}
			},
		},
		{
			conf: &RawRequestConfig{
				Method:  "POST",
				Path:    "/{{payload}}",
				Headers: make(map[string]string),
			},
			checkValue: func(r *http.Request) {
				got := r.URL.Path
				expected := "/" + testPayload

				if !(got == expected) {
					t.Errorf("test failed, got %s, expected %s", got, expected)
				}
			},
		},
		{
			conf: &RawRequestConfig{
				Method:  "GET",
				Path:    "/test?a={{payload}}",
				Headers: make(map[string]string),
			},
			checkValue: func(r *http.Request) {
				got := r.URL.Query().Encode()
				expected := "a=" + testPayload

				if !(got == expected) {
					t.Errorf("test failed, got %s, expected %s", got, expected)
				}
			},
		},
		{
			conf: &RawRequestConfig{
				Method: "GET",
				Path:   "/",
				Headers: map[string]string{
					"X-Test-Header": "Test Header {{payload}}",
				},
			},
			checkValue: func(r *http.Request) {
				got := r.Header.Get("X-Test-Header")
				expected := "Test Header " + testPayload

				if !(got == expected) {
					t.Errorf("test failed, got %s, expected %s", got, expected)
				}
			},
		},
		{
			conf: &RawRequestConfig{
				Method:  "POST",
				Path:    "/",
				Headers: make(map[string]string),
				Body:    "Test body {{payload}}",
			},
			checkValue: func(r *http.Request) {
				defer r.Body.Close()
				b, _ := io.ReadAll(r.Body)

				got := string(b)
				expected := "Test body " + testPayload

				if !(got == expected) {
					t.Errorf("test failed, got %s, expected %s", got, expected)
				}
			},
		},
		{
			conf: &RawRequestConfig{
				Method: "POST",
				Path:   "/",
				Headers: map[string]string{
					"Content-Type": "multipart/form-data; boundary=boundary",
				},
				Body: `--boundary
Content-disposition: form-data; name="field1"

Test
--boundary
Content-disposition: form-data; name="field2"
Content-Type: text/plain; charset=utf-7

Knock knock.
{{payload}}
--boundary--`,
			},
			checkValue: func(r *http.Request) {
				err := r.ParseMultipartForm(0)
				if err != nil {
					t.Errorf("got error: %s", err.Error())
				}

				got := r.FormValue("field1")
				expected := "Test"

				if !(got == expected) {
					t.Errorf("test failed, got %s, expected %s", got, expected)
				}

				got = r.FormValue("field2")
				expected = "Knock knock.\n" + testPayload

				if !(got == expected) {
					t.Errorf("test failed, got %s, expected %s", got, expected)
				}
			},
		},
	}

	for _, test := range tests {
		req, err := DefaultRawRequest.CreateRequest(url, testPayload, test.conf)
		if err != nil {
			t.Errorf("got an error: %s", err.Error())
		}

		test.checkValue(req)
	}
}
