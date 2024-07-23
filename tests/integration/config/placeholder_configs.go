package config

import (
	"io"
	"net/http"
	"strings"

	"github.com/wallarm/gotestwaf/internal/payload/placeholder"
)

var GraphQLConfigs = map[string]*struct {
	Config         *placeholder.GraphQLConfig
	Encoders       []string
	GetPayloadFunc func(r *http.Request) string
}{
	"graphql-set1": {
		Config:   &placeholder.GraphQLConfig{Method: "GET"},
		Encoders: []string{"URL"},
		GetPayloadFunc: func(r *http.Request) string {
			return r.URL.Query().Get("query")
		},
	},
	"rawrequest-set2": {
		Config:   &placeholder.GraphQLConfig{Method: "POST"},
		Encoders: []string{"Plain"},
		GetPayloadFunc: func(r *http.Request) string {
			defer r.Body.Close()
			b, _ := io.ReadAll(r.Body)
			return string(b)
		},
	},
}

var RawRequestConfigs = map[string]*struct {
	Config         *placeholder.RawRequestConfig
	Encoders       []string
	GetPayloadFunc func(r *http.Request) string
}{
	"rawrequest-set1": {
		Config:   &placeholder.RawRequestConfig{Method: "POST", Path: "/{{payload}}", Headers: map[string]string{}, Body: ""},
		Encoders: []string{"Base64", "Base64Flat", "URL"},
		GetPayloadFunc: func(r *http.Request) string {
			return strings.TrimPrefix(r.URL.Path, "/")
		},
	},
	"rawrequest-set2": {
		Config:   &placeholder.RawRequestConfig{Method: "POST", Path: "/", Headers: map[string]string{"X-Test": "{{payload}}"}, Body: ""},
		Encoders: []string{"Base64", "Base64Flat", "JSUnicode", "Plain", "URL", "XMLEntity"},
		GetPayloadFunc: func(r *http.Request) string {
			return r.Header.Get("X-Test")
		},
	},
	"rawrequest-set3": {
		Config:   &placeholder.RawRequestConfig{Method: "POST", Path: "/", Headers: map[string]string{}, Body: "{{payload}}"},
		Encoders: []string{"Base64", "Base64Flat", "JSUnicode", "Plain", "URL", "XMLEntity"},
		GetPayloadFunc: func(r *http.Request) string {
			defer r.Body.Close()
			b, _ := io.ReadAll(r.Body)
			return string(b)
		},
	},
}
