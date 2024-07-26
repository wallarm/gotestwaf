package graphql

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/config"
	"github.com/wallarm/gotestwaf/internal/helpers"
	"github.com/wallarm/gotestwaf/internal/payload"
	"github.com/wallarm/gotestwaf/internal/payload/placeholder"
	"github.com/wallarm/gotestwaf/internal/scanner/clients"
	"github.com/wallarm/gotestwaf/internal/scanner/types"
	"github.com/wallarm/gotestwaf/pkg/dnscache"
)

// List of the possible GraphQL endpoints on URL.
var checkAvailabilityEndpoints = []string{
	"/graphql",
	"/_graphql",
	"/api/graphql",
	"/GraphQL",
}

var redirectFunc func(req *http.Request, via []*http.Request) error

var _ clients.GraphQLClient = (*Client)(nil)

type Client struct {
	client     *http.Client
	headers    map[string]string
	hostHeader string

	graphqlUrl string
	httpUrl    string

	isGraphQLAvailable bool
}

func NewClient(cfg *config.Config, dnsResolver *dnscache.Resolver) (*Client, error) {
	tr := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: !cfg.TLSVerify},
		IdleConnTimeout:     time.Duration(cfg.IdleConnTimeout) * time.Second,
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConns, // net.http hardcodes DefaultMaxIdleConnsPerHost to 2!
	}

	if dnsResolver != nil {
		tr.DialContext = dnscache.DialFunc(dnsResolver, nil)
	}

	if cfg.Proxy != "" {
		proxyURL, err := url.Parse(cfg.Proxy)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't parse proxy URL")
		}

		tr.Proxy = http.ProxyURL(proxyURL)
	}

	redirectFunc = func(req *http.Request, via []*http.Request) error {
		// if maxRedirects is equal to 0 then tell the HTTP client to use
		// the first HTTP response (disable following redirects)
		if cfg.MaxRedirects == 0 {
			return http.ErrUseLastResponse
		}

		if len(via) > cfg.MaxRedirects {
			return errors.New("max redirect number exceeded")
		}

		return nil
	}

	client := &http.Client{
		Transport:     tr,
		CheckRedirect: redirectFunc,
	}

	configuredHeaders := helpers.DeepCopyMap(cfg.HTTPHeaders)
	customHeader := strings.SplitN(cfg.AddHeader, ":", 2)
	if len(customHeader) > 1 {
		header := strings.TrimSpace(customHeader[0])
		value := strings.TrimSpace(customHeader[1])
		configuredHeaders[header] = value
	}

	return &Client{
		client:     client,
		headers:    configuredHeaders,
		hostHeader: configuredHeaders["Host"],

		graphqlUrl: cfg.GraphQLURL,
		httpUrl:    cfg.URL,

		isGraphQLAvailable: true,
	}, nil
}

func (c *Client) CheckAvailability(ctx context.Context) (bool, error) {
	endpointsToCheck := checkAvailabilityEndpoints

	c.isGraphQLAvailable = false

	endpointURL, _ := url.Parse(c.graphqlUrl)

	// Add query parameter to trigger GraphQL
	queryParams := endpointURL.Query()
	queryParams.Set("query", "{__typename}")
	endpointURL.RawQuery = queryParams.Encode()

	// If cfg.GraphQLURL is different from cfg.URL, we only need to check
	// one endpoint - cfg.GraphQLURL
	if c.graphqlUrl != c.httpUrl {
		endpointsToCheck = []string{endpointURL.Path}
		endpointURL.Path = ""
	}

	for _, endpoint := range endpointsToCheck {
		endpointURL.Path = endpoint

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL.String(), nil)
		if err != nil {
			return false, errors.New("couldn't create request to check GraphQL availability")
		}

		resp, err := c.client.Do(req)
		if err != nil {
			return false, errors.New("couldn't send request to check GraphQL availability")
		}

		if resp.StatusCode == http.StatusOK {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return false, errors.Wrap(err, "couldn't read response body")
			}

			ok, err := checkAnswer(bodyBytes)
			if err != nil {
				return false, errors.Wrap(err, "couldn't check response")
			}

			// If we found correct GraphQL endpoint, save it
			if ok {
				endpointURL.RawQuery = ""

				c.graphqlUrl = endpointURL.String()
				c.isGraphQLAvailable = true

				return true, nil
			}
		}

		resp.Body.Close()
	}

	return false, nil
}

func (c *Client) IsAvailable() bool {
	return c.isGraphQLAvailable
}

func (c *Client) SendPayload(ctx context.Context, payloadInfo *payload.PayloadInfo) (types.Response, error) {
	request, err := payloadInfo.GetRequest(c.graphqlUrl, types.GoHTTPClient)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't prepare request")
	}

	r, ok := request.(*types.GoHTTPRequest)
	if !ok {
		return nil, errors.Errorf("bad request type: %T, expected %T", request, &types.GoHTTPRequest{})
	}

	req := r.Req.WithContext(ctx)

	isUAPlaceholder := payloadInfo.PlaceholderName == placeholder.DefaultUserAgent.GetName()

	for header, value := range c.headers {
		// Skip setting the User-Agent header to the value from the GoTestWAF config file
		// if the placeholder is UserAgent.
		if strings.EqualFold(header, placeholder.UAHeader) && isUAPlaceholder {
			continue
		}

		// Do not replace header values for RawRequest headers
		if req.Header.Get(header) == "" {
			req.Header.Set(header, value)
		}
	}
	req.Host = c.hostHeader

	if payloadInfo.DebugHeaderValue != "" {
		req.Header.Set(clients.GTWDebugHeader, payloadInfo.DebugHeaderValue)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "sending http request")
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading response body")
	}

	// body reuse
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	statusCode := resp.StatusCode

	reasonIndex := strings.Index(resp.Status, " ")
	reason := resp.Status[reasonIndex+1:]

	response := &types.ResponseMeta{
		StatusCode:   statusCode,
		StatusReason: reason,
		Headers:      resp.Header,
		Content:      bodyBytes,
	}

	return response, nil
}

// checkAnswer checks that answer contains "__typename" in the response body.
// Example of correct answer:
//
//	{
//	  "data": {
//	    "__typename": "Query"
//	  }
//	}
func checkAnswer(body []byte) (bool, error) {
	jsonMap := make(map[string]any)

	err := json.Unmarshal(body, &jsonMap)
	if err != nil {
		return false, errors.Wrap(err, "couldn't unmarshal JSON")
	}

	data, ok := jsonMap["data"]
	if !ok {
		return false, nil
	}

	dataMap, ok := data.(map[string]any)
	if !ok {
		return false, nil
	}

	_, ok = dataMap["__typename"]
	if ok {
		return true, nil
	}

	return false, nil
}
