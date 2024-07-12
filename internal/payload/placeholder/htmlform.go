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
)

var _ Placeholder = (*HTMLForm)(nil)

var DefaultHTMLForm = &HTMLForm{name: "HTMLForm"}

type HTMLForm struct {
	name string
}

func (p *HTMLForm) NewPlaceholderConfig(map[any]any) (PlaceholderConfig, error) {
	return nil, nil
}

func (p *HTMLForm) GetName() string {
	return p.name
}

func (p *HTMLForm) CreateRequest(requestURL, payload string, config PlaceholderConfig, httpClientType types.HTTPClientType) (types.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	randomName, err := RandomHex(Seed)
	if err != nil {
		return nil, err
	}

	bodyPayload := randomName + "=" + payload

	switch httpClientType {
	case types.GoHTTPClient:
		return p.prepareGoHTTPClientRequest(reqURL.String(), bodyPayload, config)
	case types.ChromeHTTPClient:
		return p.prepareChromeHTTPClientRequest(reqURL.String(), bodyPayload, config)
	default:
		return nil, types.NewUnknownHTTPClientError(httpClientType)
	}
}

func (p *HTMLForm) prepareGoHTTPClientRequest(requestURL, payload string, config PlaceholderConfig) (*types.GoHTTPRequest, error) {
	req, err := http.NewRequest(http.MethodPost, requestURL, strings.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	return &types.GoHTTPRequest{Req: req}, nil
}

func (p *HTMLForm) prepareChromeHTTPClientRequest(requestURL, payload string, config PlaceholderConfig) (*types.ChromeDPTasks, error) {
	reqOptions := &helpers.RequestOptions{
		Method: http.MethodPost,
		Headers: map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
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
