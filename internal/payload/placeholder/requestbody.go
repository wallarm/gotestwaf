package placeholder

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/wallarm/gotestwaf/internal/scanner/clients/chrome/helpers"

	"github.com/chromedp/chromedp"

	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

var _ Placeholder = (*RequestBody)(nil)

var DefaultRequestBody = &RequestBody{name: "RequestBody"}

type RequestBody struct {
	name string
}

func (p *RequestBody) NewPlaceholderConfig(map[any]any) (PlaceholderConfig, error) {
	return nil, nil
}

func (p *RequestBody) GetName() string {
	return p.name
}

func (p *RequestBody) CreateRequest(requestURL, payload string, config PlaceholderConfig, httpClientType types.HTTPClientType) (types.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	switch httpClientType {
	case types.GoHTTPClient:
		return p.prepareGoHTTPClientRequest(reqURL.String(), payload, config)
	case types.ChromeHTTPClient:
		return p.prepareChromeHTTPClientRequest(reqURL.String(), payload, config)
	default:
		return nil, types.NewUnknownHTTPClientError(httpClientType)
	}
}

func (p *RequestBody) prepareGoHTTPClientRequest(requestURL, payload string, config PlaceholderConfig) (*types.GoHTTPRequest, error) {
	// check if we need to set Content-Length manually here
	req, err := http.NewRequest(http.MethodPost, requestURL, strings.NewReader(payload))
	if err != nil {
		return nil, err
	}

	return &types.GoHTTPRequest{Req: req}, nil
}

func (p *RequestBody) prepareChromeHTTPClientRequest(requestURL, payload string, config PlaceholderConfig) (*types.ChromeDPTasks, error) {
	reqOptions := &helpers.RequestOptions{
		Method: http.MethodPost,
		Body:   fmt.Sprintf(`"%s"`, template.JSEscaper(payload)),
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
