package placeholder

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/chromedp/chromedp"
	"github.com/wallarm/gotestwaf/internal/scanner/clients/chrome/helpers"

	"github.com/wallarm/gotestwaf/internal/scanner/types"

	"github.com/wallarm/gotestwaf/internal/payload/encoder"
)

const jsonRequestPayloadWrapper = `{"test": true, "%s": "%s"}`

var _ Placeholder = (*JSONRequest)(nil)

var DefaultJSONRequest = &JSONRequest{name: "JSONRequest"}

type JSONRequest struct {
	name string
}

func (p *JSONRequest) NewPlaceholderConfig(map[any]any) (PlaceholderConfig, error) {
	return nil, nil
}

func (p *JSONRequest) GetName() string {
	return p.name
}

func (p *JSONRequest) CreateRequest(requestURL, payload string, config PlaceholderConfig, httpClientType types.HTTPClientType) (types.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	param, err := RandomHex(Seed)
	if err != nil {
		return nil, err
	}

	encodedPayload, err := encoder.Apply("JSUnicode", payload)
	if err != nil {
		return nil, err
	}

	jsonPayload := fmt.Sprintf(jsonRequestPayloadWrapper, param, encodedPayload)

	switch httpClientType {
	case types.GoHTTPClient:
		return p.prepareGoHTTPClientRequest(reqURL.String(), jsonPayload, config)
	case types.ChromeHTTPClient:
		return p.prepareChromeHTTPClientRequest(reqURL.String(), jsonPayload, config)
	default:
		return nil, types.NewUnknownHTTPClientError(httpClientType)
	}
}

func (p *JSONRequest) prepareGoHTTPClientRequest(requestURL, payload string, config PlaceholderConfig) (*types.GoHTTPRequest, error) {
	req, err := http.NewRequest(http.MethodPost, requestURL, strings.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	return &types.GoHTTPRequest{Req: req}, nil
}

func (p *JSONRequest) prepareChromeHTTPClientRequest(requestURL, payload string, config PlaceholderConfig) (*types.ChromeDPTasks, error) {
	reqOptions := &helpers.RequestOptions{
		Method: http.MethodPost,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: fmt.Sprintf(`"%s"`, template.JSEscaper(payload)),
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
