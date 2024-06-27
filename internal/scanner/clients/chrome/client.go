package chrome

import (
	"context"
	"strings"
	"sync"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/wallarm/gotestwaf/internal/config"
	"github.com/wallarm/gotestwaf/internal/helpers"
	"github.com/wallarm/gotestwaf/internal/payload"
	"github.com/wallarm/gotestwaf/internal/scanner/clients"
	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

var _ clients.HTTPClient = (*Client)(nil)

var DefaultChromeDPExecAllocatorOptions = append(
	chromedp.DefaultExecAllocatorOptions[:],
	// Disable the CORS policy constraints
	chromedp.Flag("disable-web-security", true),
)

type Client struct {
	execAllocatorOptions []chromedp.ExecAllocatorOption
	disableLogs          bool

	headers map[string]string
}

func NewClient(cfg *config.Config) (*Client, error) {
	execAllocatorOptions := DefaultChromeDPExecAllocatorOptions[:]

	disableLogs := false
	logLevel, _ := logrus.ParseLevel(cfg.LogLevel)
	if logLevel < logrus.DebugLevel {
		disableLogs = true
	}

	if cfg.Proxy != "" {
		execAllocatorOptions = append(
			execAllocatorOptions,
			chromedp.ProxyServer(cfg.Proxy),
			// By default, Chrome will bypass localhost.
			// The test server is bound to localhost, so we should add the
			// following flag to use the proxy for localhost URLs.
			chromedp.Flag("proxy-bypass-list", "<-loopback>"),
		)
	}

	if !cfg.TLSVerify {
		execAllocatorOptions = append(
			execAllocatorOptions,
			chromedp.Flag("ignore-certificate-errors", "1"),
			chromedp.Flag("allow-insecure-localhost", "1"),
		)
	}

	configuredHeaders := helpers.DeepCopyMap(cfg.HTTPHeaders)
	for k := range configuredHeaders {
		if strings.EqualFold(k, "host") {
			delete(configuredHeaders, k)
			break
		}
	}

	customHeader := strings.SplitN(cfg.AddHeader, ":", 2)
	if len(customHeader) > 1 {
		header := strings.TrimSpace(customHeader[0])
		value := strings.TrimSpace(customHeader[1])
		configuredHeaders[header] = value
	}

	c := &Client{
		execAllocatorOptions: execAllocatorOptions,
		disableLogs:          disableLogs,
		headers:              configuredHeaders,
	}

	return c, nil
}

func (c *Client) SendPayload(
	ctx context.Context,
	targetURL string,
	payloadInfo *payload.PayloadInfo,
) (types.Response, error) {
	request, err := payloadInfo.GetRequest(targetURL, types.ChromeHTTPClient)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't prepare request")
	}

	r, ok := request.(*types.ChromeDPTasks)
	if !ok {
		return nil, errors.Errorf("bad request type: %T, expected %T", request, &types.ChromeDPTasks{})
	}

	// Create a new Chrome allocator context
	allocCtx, allocCtxCancel := chromedp.NewExecAllocator(ctx, c.execAllocatorOptions...)
	defer allocCtxCancel()

	var logOptions []chromedp.ContextOption
	if c.disableLogs {
		logOptions = append(
			logOptions,
			chromedp.WithLogf(discardLogs),
			chromedp.WithDebugf(discardLogs),
			chromedp.WithErrorf(discardLogs),
		)
	}

	// Create a new Chrome context
	chromeCtx, chromeCtxCancel := chromedp.NewContext(allocCtx, logOptions...)
	defer chromeCtxCancel()

	headers := make(network.Headers)
	for k, v := range c.headers {
		if strings.EqualFold(k, "host") {
			continue
		}

		headers[k] = v
	}

	var wg sync.WaitGroup
	errorChan := make(chan error, 10)

	wg.Add(1)
	go func() {
		defer wg.Done()

		// Get home page
		tasks := chromedp.Tasks{chromedp.Navigate(targetURL)}
		if len(headers) > 0 {
			tasks = append(chromedp.Tasks{network.SetExtraHTTPHeaders(headers)}, tasks...)
		}

		if err := chromedp.Run(chromeCtx, tasks); err != nil {
			errorChan <- errors.Wrap(err, "failed to execute Chrome tasks")
		}

		// Perform request with payload
		if payloadInfo.DebugHeaderValue != "" {
			headers[clients.GTWDebugHeader] = payloadInfo.DebugHeaderValue
		}

		for k, v := range r.UserAgentHeader {
			headers[k] = v
		}

		tasks = chromedp.Tasks{}
		if len(headers) > 0 {
			tasks = chromedp.Tasks{network.SetExtraHTTPHeaders(headers)}
		}
		tasks = append(tasks, r.Tasks...)

		if err := chromedp.Run(chromeCtx, tasks); err != nil {
			errorChan <- errors.Wrap(err, "failed to execute Chrome tasks")
		}

		close(errorChan)
	}()

	err = nil

	// Collect errors
forLoop:
	for {
		select {
		case e, ok := <-errorChan:
			if !ok {
				break forLoop
			}
			err = multierror.Append(err, e)
		}
	}

	// Wait Chrome-related goroutines
	wg.Wait()

	if err != nil {
		return nil, err
	}

	return r.ResponseMeta, nil
}

func (c *Client) SendRequest(
	ctx context.Context,
	req types.Request,
) (types.Response, error) {
	r, ok := req.(*types.ChromeDPTasks)
	if !ok {
		return nil, errors.Errorf("bad request type: %T, expected %T", req, &types.ChromeDPTasks{})
	}

	// Create a new Chrome allocator context
	allocCtx, allocCtxCancel := chromedp.NewExecAllocator(ctx, c.execAllocatorOptions...)
	defer allocCtxCancel()

	var logOptions []chromedp.ContextOption
	if c.disableLogs {
		logOptions = append(
			logOptions,
			chromedp.WithLogf(discardLogs),
			chromedp.WithDebugf(discardLogs),
			chromedp.WithErrorf(discardLogs),
		)
	}

	// Create a new Chrome context
	chromeCtx, chromeCtxCancel := chromedp.NewContext(allocCtx, logOptions...)
	defer chromeCtxCancel()

	headers := make(network.Headers)
	for k, v := range c.headers {
		if strings.EqualFold(k, "host") {
			continue
		}

		headers[k] = v
	}

	if r.DebugHeaderValue != "" {
		headers[clients.GTWDebugHeader] = r.DebugHeaderValue
	}

	var tasks chromedp.Tasks
	if len(headers) > 0 {
		tasks = append(chromedp.Tasks{network.SetExtraHTTPHeaders(headers)}, tasks...)
	}
	tasks = append(tasks, r.Tasks...)

	var err error
	var wg sync.WaitGroup
	errorChan := make(chan error, 10)

	// Hold the latest response information
	var latestResponse *types.ResponseMeta
	var mu sync.Mutex

	// Enable Network domain and set request interception
	if err = chromedp.Run(chromeCtx, network.Enable()); err != nil {
		return nil, errors.Wrap(err, "couldn't enable network domain")
	}

	// Listen for network events
	chromedp.ListenTarget(chromeCtx, func(ev interface{}) {
		if ev, ok := ev.(*network.EventResponseReceived); ok {
			wg.Add(1)
			go func() {
				defer wg.Done()

				mu.Lock()
				defer mu.Unlock()

				localCtx := chromedp.FromContext(chromeCtx)
				executor := cdp.WithExecutor(chromeCtx, localCtx.Target)

				// Get the response body
				body, _ := network.GetResponseBody(ev.RequestID).Do(executor)

				response := ev.Response
				info := &types.ResponseMeta{
					StatusCode:   int(response.Status),
					StatusReason: response.StatusText,
					Headers:      headersToMap(response.Headers),
					Content:      body,
				}

				// Update the latest response
				latestResponse = info
			}()
		}
	})

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := chromedp.Run(chromeCtx, r.Tasks); err != nil {
			errorChan <- errors.Wrap(err, "failed to execute Chrome tasks")
		}

		close(errorChan)
	}()

	err = nil

	// Collect errors
forLoop:
	for {
		select {
		case e, ok := <-errorChan:
			if !ok {
				break forLoop
			}
			err = multierror.Append(err, e)
		}
	}

	// Wait Chrome-related goroutines
	wg.Wait()

	if err != nil {
		return nil, err
	}

	return latestResponse, nil
}

// discardLogs serves as a no-op logging function for chromedp
// to suppress all internal logging output.
func discardLogs(string, ...interface{}) {}
