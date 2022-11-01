package report

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/pkg/report"
)

const serverURL = "https://gotestwaf.wallarm.tools/v1/send-report"

var _ error = (*ErrorResponse)(nil)

type ErrorResponse struct {
	Msg string `json:"msg"`
}

func (e *ErrorResponse) Error() string {
	return e.Msg
}

func sendEmail(ctx context.Context, reportData *report.HtmlReport, email string) error {
	requestUrl, err := url.Parse(serverURL)
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
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil

	case http.StatusBadRequest,
		http.StatusRequestEntityTooLarge,
		http.StatusTooManyRequests,
		http.StatusInternalServerError:

		var errResp ErrorResponse

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "couldn't read error message from server")
		}

		if err := json.Unmarshal(body, &errResp); err != nil {
			return errors.Wrap(err, "couldn't parse error message from server")
		}

		return &errResp

	default:
		return fmt.Errorf("bad status code: %d", resp.StatusCode)
	}
}
