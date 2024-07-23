package placeholder

import (
	"crypto/sha256"
	"net/http"
	"net/url"
	"strings"

	"github.com/wallarm/gotestwaf/internal/scanner/types"

	"github.com/pkg/errors"
)

type GraphQL struct {
	name string
}

type GraphQLConfig struct {
	Method string
}

var DefaultGraphQL = &GraphQL{name: "GraphQL"}

var _ Placeholder = (*GraphQL)(nil)

func (p *GraphQL) NewPlaceholderConfig(conf map[any]any) (PlaceholderConfig, error) {
	result := &GraphQLConfig{}

	method, ok := conf["method"]
	if !ok {
		return nil, &BadPlaceholderConfigError{
			name: p.name,
			err:  errors.New("empty method"),
		}
	}
	result.Method, ok = method.(string)
	if !ok {
		return nil, &BadPlaceholderConfigError{
			name: p.name,
			err:  errors.Errorf("unknown type of 'method' field, expected string, got %T", method),
		}
	}

	switch result.Method {
	case http.MethodGet, http.MethodPost:
		return result, nil

	default:
		return nil, &BadPlaceholderConfigError{
			name: p.name,
			err:  errors.Errorf("unknown HTTP method, expected GET or POST, got %T", result.Method),
		}
	}
}

func (p *GraphQL) GetName() string {
	return p.name
}

func (p *GraphQL) CreateRequest(requestURL, payload string, config PlaceholderConfig, httpClientType types.HTTPClientType) (types.Request, error) {
	if httpClientType != types.GoHTTPClient {
		return nil, errors.New("CreateRequest only support GoHTTPClient")
	}

	conf, ok := config.(*GraphQLConfig)
	if !ok {
		return nil, &BadPlaceholderConfigError{
			name: p.name,
			err:  errors.Errorf("bad config type: got %T, expected: %T", config, &GraphQLConfig{}),
		}
	}

	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	reqest := &types.GoHTTPRequest{}

	switch conf.Method {
	case http.MethodGet:
		queryParams := reqURL.Query()
		queryParams.Set("query", payload)
		reqURL.RawQuery = queryParams.Encode()

		req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
		if err != nil {
			return nil, err
		}

		reqest.Req = req

		return reqest, nil

	case http.MethodPost:
		req, err := http.NewRequest(http.MethodPost, reqURL.String(), strings.NewReader(payload))
		if err != nil {
			return nil, err
		}

		reqest.Req = req

		return reqest, nil

	default:
		return nil, &BadPlaceholderConfigError{
			name: p.name,
			err:  errors.Errorf("unknown HTTP method, expected GET or POST, got %T", conf.Method),
		}
	}
}

func (g *GraphQLConfig) Hash() []byte {
	sha256sum := sha256.New()
	sha256sum.Write([]byte(g.Method))
	return sha256sum.Sum(nil)
}
