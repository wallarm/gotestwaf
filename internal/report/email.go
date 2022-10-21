package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/pkg/report"
)

const server = "http://localhost:3000/send-email"

func sendEmail(reportData *report.HtmlReport, email string) error {
	requestUrl, err := url.Parse(server)
	if err != nil {
		return errors.Wrap(err, "couldn't parse server URL")
	}

	queryParams := requestUrl.Query()
	queryParams.Set("email", email)
	requestUrl.RawQuery = queryParams.Encode()

	data, err := json.Marshal(reportData)
	if err != nil {
		return errors.Wrap(err, "couldn't marshal report data into JSON format")
	}

	req, err := http.NewRequest(http.MethodPost, requestUrl.String(), bytes.NewReader(data))
	if err != nil {
		return errors.Wrap(err, "couldn't create request")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "couldn't send request to server")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	return nil
}
