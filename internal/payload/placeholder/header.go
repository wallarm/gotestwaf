package placeholder

import (
	"net/http"
	"net/url"

	"github.com/chromedp/chromedp"
	"github.com/wallarm/gotestwaf/internal/scanner/clients/chrome/helpers"

	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

var _ Placeholder = (*Header)(nil)

var DefaultHeader = &Header{name: "Header"}

type Header struct {
	name string
}

func (p *Header) NewPlaceholderConfig(map[any]any) (PlaceholderConfig, error) {
	return nil, nil
}

func (p *Header) GetName() string {
	return p.name
}

func (p *Header) CreateRequest(requestURL, payload string, config PlaceholderConfig, httpClientType types.HTTPClientType) (types.Request, error) {
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

func (p *Header) prepareGoHTTPClientRequest(requestURL, payload string, config PlaceholderConfig) (*types.GoHTTPRequest, error) {
	randomName, err := RandomHex(Seed)
	if err != nil {
		return nil, err
	}

	randomHeader := "X-" + randomName
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add(randomHeader, payload)

	return &types.GoHTTPRequest{Req: req}, nil
}

func (p *Header) prepareChromeHTTPClientRequest(requestURL, payload string, config PlaceholderConfig) (*types.ChromeDPTasks, error) {
	randomName, err := RandomHex(Seed)
	if err != nil {
		return nil, err
	}

	randomHeader := "X-" + randomName

	reqOptions := &helpers.RequestOptions{
		Method: http.MethodGet,
		Headers: map[string]string{
			randomHeader: payload,
		},
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
