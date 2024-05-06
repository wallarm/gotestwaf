package placeholder

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/url"

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
	switch httpClientType {
	case types.GoHTTPClient:
		return p.prepareGoHTTPClientRequest(requestURL, payload, config)
	case types.ChromeHTTPClient:
		return p.prepareChromeHTTPClientRequest(requestURL, payload, config)
	default:
		return nil, types.NewUnknownHTTPClientError(httpClientType)
	}
}

func (p *HTMLMultipartForm) prepareGoHTTPClientRequest(requestURL, payload string, config PlaceholderConfig) (*types.GoHTTPRequest, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

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

	req, err := http.NewRequest("POST", reqURL.String(), bytes.NewReader(body.Bytes()))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", writer.FormDataContentType())

	return &types.GoHTTPRequest{Req: req}, nil
}

func (p *HTMLMultipartForm) prepareChromeHTTPClientRequest(requestURL, payload string, config PlaceholderConfig) (*types.ChromeDPTasks, error) {
	return nil, nil
}
