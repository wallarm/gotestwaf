package placeholder

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/chromedp/chromedp"
	"github.com/wallarm/gotestwaf/internal/scanner/clients/chrome/helpers"

	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

var _ Placeholder = (*URLParam)(nil)

var DefaultURLParam = &URLParam{name: "URLParam"}

type URLParam struct {
	name string
}

func (p *URLParam) NewPlaceholderConfig(map[any]any) (PlaceholderConfig, error) {
	return nil, nil
}

func (p *URLParam) GetName() string {
	return p.name
}

func (p *URLParam) CreateRequest(requestURL, payload string, config PlaceholderConfig, httpClientType types.HTTPClientType) (types.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	reqURL.Fragment = ""
	urlWithPayload := reqURL.String()
	if reqURL.RawQuery == "" {
		for i := len(urlWithPayload) - 1; i >= 0; i-- {
			if urlWithPayload[i] != '/' {
				if strings.HasSuffix(reqURL.Path, urlWithPayload[i:]) {
					urlWithPayload = urlWithPayload[:i+1] + "?"
				} else {
					urlWithPayload = urlWithPayload[:i+1] + "/?"
				}
				break
			}
		}
	} else {
		urlWithPayload += "&"
	}

	param, err := RandomHex(Seed)
	if err != nil {
		return nil, err
	}

	urlWithPayload += param + "="

	switch httpClientType {
	case types.GoHTTPClient:
		return p.prepareGoHTTPClientRequest(urlWithPayload, payload, config)
	case types.ChromeHTTPClient:
		return p.prepareChromeHTTPClientRequest(urlWithPayload, payload, config)
	default:
		return nil, types.NewUnknownHTTPClientError(httpClientType)
	}
}

func (p *URLParam) prepareGoHTTPClientRequest(requestURL, payload string, config PlaceholderConfig) (*types.GoHTTPRequest, error) {
	requestURL += payload

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	return &types.GoHTTPRequest{Req: req}, err
}

func (p *URLParam) prepareChromeHTTPClientRequest(requestURL, payload string, config PlaceholderConfig) (*types.ChromeDPTasks, error) {
	jsEncodedPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	jsEncodedPayloadStr := strings.Trim(string(jsEncodedPayload), "\"")

	requestURL += jsEncodedPayloadStr

	reqOptions := &helpers.RequestOptions{
		Method: http.MethodGet,
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
