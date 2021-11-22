package placeholder

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/url"
)

func HTMLMultipartForm(requestURL, payload string) (*http.Request, error) {
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

	return req, nil
}
