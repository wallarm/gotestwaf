package report

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/pkg/report"
)

const server = "https://gotestwaf.wallarm.tools/v1/send-email"

var _ error = (*ErrorResponse)(nil)

type ErrorResponse struct {
	Msg string `json:"msg"`
}

func (e *ErrorResponse) Error() string {
	return e.Msg
}

func sendEmail(ctx context.Context, reportData *report.HtmlReport, email string) error {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestUrl.String(), bytes.NewReader(data))
	if err != nil {
		return errors.Wrap(err, "couldn't create request")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "couldn't send request to server")
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return nil

	case http.StatusInternalServerError:
		return fmt.Errorf("bad status code: %d", resp.StatusCode)

	default:
		var errResp ErrorResponse
		var body []byte

		resp.Body.Read(body)

		if err := json.Unmarshal(body, &errResp); err != nil {
			return errors.Wrap(err, "couldn't parse error message from server")
		}

		return &errResp
	}
}
