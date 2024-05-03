package placeholder

import (
	"net/http"
	"net/url"
	"strings"

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
	req, err := http.NewRequest("POST", reqURL.String(), strings.NewReader(bodyPayload))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	return &types.GoHTTPRequest{Req: req}, nil
}
