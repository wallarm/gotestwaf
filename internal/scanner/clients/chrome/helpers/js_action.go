package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"net/http"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

// jsCodeTemplate is a JavaScript code to send an HTTP request and
// return the response details.
const jsCodeTemplate = `
const f = async function() {
	let err = null;

	const response = await fetch(
		'{{.URL}}',
		{
			method: "{{.Method}}",
			{{if .Headers}}headers: {{.Headers}},{{end}}
			{{if .Body}}body: {{.Body}}{{end}}
		}
	).catch(e => {
		err = {
			Error: e.message,
		};
	});

	if (err) {
		return err;
	}

	const headers = {};
	response.headers.forEach((value, key) => {
		headers[key] = value;
	});

	const bodyBuffer = await response.arrayBuffer();
	const body = Array.from(new Uint8Array(bodyBuffer));

	return {
		StatusCode: response.status,
		StatusText: response.statusText,
		Headers: headers,
		Content: body
	};
};
window.returnValue = f();
`

// RequestOptions represents fetch options.
type RequestOptions struct {
	Method  string
	Headers map[string]string
	Body    string
}

type options struct {
	URL     string
	Method  string
	Headers template.HTML
	Body    template.HTML
}

// response represents data received from JS script.
type response struct {
	StatusCode   int
	StatusReason string
	Headers      map[string]string
	Content      []byte
	Error        string
}

func GetFetchRequest(targetURL string, reqOptions *RequestOptions) (chromedp.Action, *types.ResponseMeta, error) {
	if reqOptions == nil {
		return nil, nil, errors.New("no request options provided")
	}

	if reqOptions.Method == "" {
		return nil, nil, errors.New("request method is empty")
	}

	opts := &options{
		URL:    targetURL,
		Method: reqOptions.Method,
	}

	if reqOptions.Headers != nil && len(reqOptions.Headers) > 0 {
		headersJSON, err := json.Marshal(reqOptions.Headers)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to marshal headers")
		}

		opts.Headers = template.HTML(headersJSON)
	}

	if reqOptions.Body != "" {
		opts.Body = template.HTML(reqOptions.Body)
	}

	t := template.Must(template.New("jsCodeTemplate").Parse(jsCodeTemplate))

	jsCode := bytes.NewBuffer(nil)
	if err := t.Execute(jsCode, opts); err != nil {
		return nil, nil, errors.Wrap(err, "couldn't create JS snippet")
	}

	responseMeta := &types.ResponseMeta{}
	responseMetaRaw := &response{}

	f := chromedp.ActionFunc(func(ctx context.Context) error {
		// Run the JavaScript and capture the result
		var jsResult []byte
		err := chromedp.Evaluate(
			jsCode.String(),
			&jsResult,
			func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
				return p.WithAwaitPromise(true)
			},
		).Do(ctx)
		if err != nil {
			return err
		}

		// Parse the JSON result into the provided result interface
		err = json.Unmarshal(jsResult, responseMetaRaw)
		if err != nil {
			return err
		}

		responseMeta.Headers = make(http.Header)
		responseMeta.StatusCode = responseMetaRaw.StatusCode
		responseMeta.StatusReason = responseMetaRaw.StatusReason
		responseMeta.Content = responseMetaRaw.Content
		responseMeta.Error = responseMetaRaw.Error

		for k, v := range responseMetaRaw.Headers {
			responseMeta.Headers[k] = []string{v}
		}

		return nil
	})

	return f, responseMeta, nil
}
