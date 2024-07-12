package placeholder

import (
	"bytes"
	"fmt"
	"html/template"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/chromedp/chromedp"
	"github.com/wallarm/gotestwaf/internal/scanner/clients/chrome/helpers"

	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

var _ Placeholder = (*HTMLMultipartForm)(nil)

var DefaultHTMLMultipartForm = &HTMLMultipartForm{name: "HTMLMultipartForm"}

type HTMLMultipartForm struct {
	name string
}

func (p *HTMLMultipartForm) NewPlaceholderConfig(map[any]any) (PlaceholderConfig, error) {
	return nil, nil
}

func (p *HTMLMultipartForm) GetName() string {
	return p.name
}

func (p *HTMLMultipartForm) CreateRequest(requestURL, payload string, config PlaceholderConfig, httpClientType types.HTTPClientType) (types.Request, error) {
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

func (p *HTMLMultipartForm) prepareGoHTTPClientRequest(requestURL, payload string, config PlaceholderConfig) (*types.GoHTTPRequest, error) {
	randomName, err := RandomHex(Seed)
	if err != nil {
		return nil, err
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	fw, err := writer.CreateFormField(randomName)
	if err != nil {
		return nil, err
	}

	_, err = fw.Write([]byte(payload))
	if err != nil {
		return nil, err
	}

	writer.Close()

	req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewReader(body.Bytes()))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", writer.FormDataContentType())

	return &types.GoHTTPRequest{Req: req}, nil
}

func (p *HTMLMultipartForm) prepareChromeHTTPClientRequest(requestURL, payload string, config PlaceholderConfig) (*types.ChromeDPTasks, error) {
	randomName, err := RandomHex(Seed)
	if err != nil {
		return nil, err
	}

	reqOptions := &helpers.RequestOptions{
		Method: http.MethodPost,
		Body: fmt.Sprintf(
			`(() => { const formData  = new FormData(); formData.append("%s", "%s"); return formData; })()`,
			randomName, template.JSEscaper(payload),
		),
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
