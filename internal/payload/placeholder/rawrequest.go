package placeholder

import (
	"crypto/sha256"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/chromedp/chromedp"
	"github.com/wallarm/gotestwaf/internal/scanner/clients/chrome/helpers"

	"github.com/wallarm/gotestwaf/internal/scanner/types"

	"github.com/pkg/errors"
)

var _ Placeholder = (*RawRequest)(nil)
var _ PlaceholderConfig = (*RawRequestConfig)(nil)

var DefaultRawRequest = &RawRequest{name: "RawRequest"}

type RawRequest struct {
	name string
}

type RawRequestConfig struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    string
}

func (p *RawRequest) NewPlaceholderConfig(conf map[any]any) (PlaceholderConfig, error) {
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

func (p *RawRequest) GetName() string {
	return p.name
}

// CreateRequest creates a new request from config.
// config must be a RawRequestConfig struct.
func (p *RawRequest) CreateRequest(requestURL, payload string, config PlaceholderConfig, httpClientType types.HTTPClientType) (types.Request, error) {
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

	requestURL = strings.ReplaceAll(requestURL, payloadPlaceholder, url.PathEscape(payload))

	switch httpClientType {
	case types.GoHTTPClient:
		return p.prepareGoHTTPClientRequest(requestURL, payload, conf)
	case types.ChromeHTTPClient:
		return p.prepareChromeHTTPClientRequest(requestURL, payload, conf)
	default:
		return nil, types.NewUnknownHTTPClientError(httpClientType)
	}
}

func (p *RawRequest) prepareGoHTTPClientRequest(requestURL, payload string, config *RawRequestConfig) (*types.GoHTTPRequest, error) {
	var bodyReader io.Reader
	body := strings.ReplaceAll(config.Body, payloadPlaceholder, payload)
	if len(body) != 0 {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequest(config.Method, requestURL, bodyReader)
	if err != nil {
		return nil, err
	}
	for k, v := range config.Headers {
		req.Header.Add(k, strings.ReplaceAll(v, payloadPlaceholder, payload))
	}

	return &types.GoHTTPRequest{Req: req}, nil
}

func (p *RawRequest) prepareChromeHTTPClientRequest(requestURL, payload string, config *RawRequestConfig) (*types.ChromeDPTasks, error) {
	headers := make(map[string]string)
	for k, v := range config.Headers {
		headers[k] = strings.ReplaceAll(v, payloadPlaceholder, payload)
	}

	body := fmt.Sprintf(`"%s"`, template.JSEscaper(strings.ReplaceAll(config.Body, payloadPlaceholder, payload)))

	reqOptions := &helpers.RequestOptions{
		Method:  config.Method,
		Headers: headers,
		Body:    body,
	}

	task, responseMeta, err := helpers.GetFetchRequest(requestURL, reqOptions)
	if err != nil {
		return nil, err
	}

	tasks := &types.ChromeDPTasks{
		Tasks:        chromedp.Tasks{task},
		ResponseMeta: responseMeta,
	}

	return tasks, nil
}

func (r *RawRequestConfig) Hash() []byte {
	sha256sum := sha256.New()

	sha256sum.Write([]byte(r.Method))
	sha256sum.Write([]byte(r.Path))

	for header, value := range r.Headers {
		sha256sum.Write([]byte(header))
		sha256sum.Write([]byte(value))
	}

	sha256sum.Write([]byte(r.Body))

	return sha256sum.Sum(nil)
}
