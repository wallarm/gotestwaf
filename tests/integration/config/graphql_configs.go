package config

import (
	"io"
	"net/http"

	"github.com/wallarm/gotestwaf/internal/payload/placeholder"
)

var GraphQLConfigs = map[string]*struct {
	Config         *placeholder.GraphQLConfig
	GetPayloadFunc func(r *http.Request) string
}{
	"graphql-set1": {
		Config: &placeholder.GraphQLConfig{Method: "GET"},
		GetPayloadFunc: func(r *http.Request) string {
			query := r.URL.Query()
			payload := query["query"][0]
			return payload
		},
	},
	"graphql-set2": {
		Config: &placeholder.GraphQLConfig{Method: "POST"},
		GetPayloadFunc: func(r *http.Request) string {
			defer r.Body.Close()
			b, _ := io.ReadAll(r.Body)
			return string(b)
		},
	},
}
