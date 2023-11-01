package placeholder

import (
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

type RawRequest struct {
	name string
}

type RawRequestConfig struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    string
}

var DefaultRawRequest = RawRequest{name: "RawRequest"}

var _ Placeholder = (*RawRequest)(nil)

func (p RawRequest) newConfig(conf map[any]any) (any, error) {
	result := &RawRequestConfig{}

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

	path, ok := conf["path"]
	if !ok {
		result.Path = "/"
	} else {
		result.Path, ok = path.(string)
		if !ok {
			return nil, &BadPlaceholderConfigError{
				name: p.name,
				err:  errors.Errorf("unknown type of 'path' field, expected string, got %T", path),
			}
		}

		if len(result.Path) == 0 {
			result.Path = "/"
		}
	}

	result.Headers = make(map[string]string)
	headers, ok := conf["headers"]
	if ok {
		typedHeaders, ok := headers.(map[any]any)
		if !ok {
			return nil, &BadPlaceholderConfigError{
				name: p.name,
				err:  errors.Errorf("unknown type of 'headers' field, expected map[string]string, got %T", typedHeaders),
			}
		}

		for k, v := range typedHeaders {
			header, okHeader := k.(string)
			value, okValue := v.(string)

			if !okHeader || !okValue {
				return nil, &BadPlaceholderConfigError{
					name: p.name,
					err:  errors.Errorf("unknown type of 'headers' field, expected map[string]string, got map[%T]%T", k, v),
				}
			}

			result.Headers[header] = value
		}
	}

	body, ok := conf["body"]
	if ok {
		result.Body, ok = body.(string)
		if !ok {
			return nil, &BadPlaceholderConfigError{
				name: p.name,
				err:  errors.Errorf("unknown type of 'body' field, expected string, got %T", body),
			}
		}
	}

	return result, nil
}

func (p RawRequest) GetName() string {
	return p.name
}

// CreateRequest creates a new request from config.
// config must be a RawRequestConfig struct.
func (p RawRequest) CreateRequest(requestURL, payload string, config any) (*http.Request, error) {
	conf, ok := config.(*RawRequestConfig)
	if !ok {
		return nil, &BadPlaceholderConfigError{
			name: p.name,
			err:  errors.Errorf("bad config type: got %T, expected: %T", config, &RawRequestConfig{}),
		}
	}

	if !strings.HasSuffix(requestURL, "/") {
		requestURL += "/"
	}

	if strings.HasPrefix(conf.Path, "/") {
		requestURL += conf.Path[1:]
	} else {
		requestURL += conf.Path
	}

	requestURL = strings.ReplaceAll(requestURL, payloadPlaceholder, payload)

	var bodyReader io.Reader
	body := strings.ReplaceAll(conf.Body, payloadPlaceholder, payload)
	if len(body) != 0 {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequest(conf.Method, requestURL, bodyReader)
	if err != nil {
		return nil, err
	}
	for k, v := range conf.Headers {
		req.Header.Add(k, strings.ReplaceAll(v, payloadPlaceholder, payload))
	}

	return req, nil
}
