package config

import (
	"io"
	"net/http"
	"strings"

	"github.com/wallarm/gotestwaf/internal/payload/placeholder"
)

var RawRequestConfigs = map[string]*struct {
	Config         *placeholder.RawRequestConfig
	GetPayloadFunc func(r *http.Request) string
}{
	"rawrequest-set1": {
		Config: &placeholder.RawRequestConfig{Method: "POST", Path: "/{{payload}}", Headers: map[string]string{}, Body: ""},
		GetPayloadFunc: func(r *http.Request) string {
			return strings.TrimPrefix(r.URL.Path, "/")
		},
	},
	"rawrequest-set2": {
		Config: &placeholder.RawRequestConfig{Method: "POST", Path: "/", Headers: map[string]string{"X-Test": "{{payload}}"}, Body: ""},
		GetPayloadFunc: func(r *http.Request) string {
			return r.Header.Get("X-Test")
		},
	},
	"rawrequest-set3": {
		Config: &placeholder.RawRequestConfig{Method: "POST", Path: "/", Headers: map[string]string{}, Body: "{{payload}}"},
		GetPayloadFunc: func(r *http.Request) string {
			defer r.Body.Close()
			b, _ := io.ReadAll(r.Body)
			return string(b)
		},
	},
}
