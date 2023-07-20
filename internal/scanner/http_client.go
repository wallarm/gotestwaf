package scanner

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/config"
	"github.com/wallarm/gotestwaf/internal/dnscache"
	"github.com/wallarm/gotestwaf/internal/payload/encoder"
	"github.com/wallarm/gotestwaf/internal/payload/placeholder"
)

const (
	GTWDebugHeader = "X-GoTestWAF-Test"

	getCookiesRepeatAttempts = 3
)

var redirectFunc func(req *http.Request, via []*http.Request) error

type HTTPClient struct {
	client     *http.Client
	headers    map[string]string
	hostHeader string

	followCookies bool
	renewSession  bool
}

func NewHTTPClient(cfg *config.Config, dnsResolver *dnscache.Resolver) (*HTTPClient, error) {
	tr := &http.Transport{
		DialContext:         dnscache.DialFunc(dnsResolver, nil),
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: !cfg.TLSVerify},
		IdleConnTimeout:     time.Duration(cfg.IdleConnTimeout) * time.Second,
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConns, // net.http hardcodes DefaultMaxIdleConnsPerHost to 2!
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

	if cfg.FollowCookies && !cfg.RenewSession {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, err
		}

		client.Jar = jar
	}

	configuredHeaders := cfg.HTTPHeaders
	customHeader := strings.SplitN(cfg.AddHeader, ":", 2)
	if len(customHeader) > 1 {
		header := strings.TrimSpace(customHeader[0])
		value := strings.TrimSpace(customHeader[1])
		configuredHeaders[header] = value
	}

	return &HTTPClient{
		client:        client,
		headers:       configuredHeaders,
		hostHeader:    configuredHeaders["Host"],
		followCookies: cfg.FollowCookies,
		renewSession:  cfg.RenewSession,
	}, nil
}

func (c *HTTPClient) SendPayload(
	ctx context.Context,
	targetURL string,
	payload string,
	encoderName string,
	placeholderName string,
	placeholderConfig any,
	testHeaderValue string,
) (
	responseMsgHeader string,
	responseBody string,
	statusCode int,
	err error,
) {
	encodedPayload, err := encoder.Apply(encoderName, payload)
	if err != nil {
		return "", "", 0, errors.Wrap(err, "encoding payload")
	}

	req, err := placeholder.Apply(targetURL, encodedPayload, placeholderName, placeholderConfig)
	if err != nil {
		return "", "", 0, errors.Wrap(err, "apply placeholder")
	}

	req = req.WithContext(ctx)

	isUAPlaceholder := placeholderName == placeholder.DefaultUserAgent.GetName()

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

	if testHeaderValue != "" {
		req.Header.Set(GTWDebugHeader, testHeaderValue)
	}

	if c.followCookies && c.renewSession {
		cookies, err := c.getCookies(ctx, targetURL)
		if err != nil {
			return "", "", 0, errors.Wrap(err, "couldn't get cookies for malicious request")
		}

		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", "", 0, errors.Wrap(err, "sending http request")
	}
	defer resp.Body.Close()

	msgHeader, err := httputil.DumpResponse(resp, false)
	if err != nil {
		return "", "", 0, errors.Wrap(err, "dumping http response")
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", 0, errors.Wrap(err, "reading response body")
	}
	statusCode = resp.StatusCode

	if c.followCookies && !c.renewSession && c.client.Jar != nil {
		c.client.Jar.SetCookies(req.URL, resp.Cookies())
	}

	return string(msgHeader), string(bodyBytes), statusCode, nil
}

func (c *HTTPClient) SendRequest(
	req *http.Request,
	testHeaderValue string,
) (
	respHeaders http.Header,
	responseMsgHeader string,
	body string,
	statusCode int,
	err error,
) {
	for header, value := range c.headers {
		req.Header.Set(header, value)
	}
	req.Host = c.hostHeader

	if testHeaderValue != "" {
		req.Header.Set(GTWDebugHeader, testHeaderValue)
	}

	if c.followCookies && c.renewSession {
		cookies, err := c.getCookies(req.Context(), GetTargetURL(req.URL))
		if err != nil {
			return nil, "", "", 0, errors.Wrap(err, "couldn't get cookies for malicious request")
		}

		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, "", "", 0, errors.Wrap(err, "sending http request")
	}
	defer resp.Body.Close()

	msgHeader, err := httputil.DumpResponse(resp, false)
	if err != nil {
		return nil, "", "", 0, errors.Wrap(err, "dumping http response")
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", "", 0, errors.Wrap(err, "reading response body")
	}
	statusCode = resp.StatusCode

	if c.followCookies && !c.renewSession && c.client.Jar != nil {
		c.client.Jar.SetCookies(req.URL, resp.Cookies())
	}

	return resp.Header, string(msgHeader), string(bodyBytes), statusCode, nil
}

func (c *HTTPClient) getCookies(ctx context.Context, targetURL string) ([]*http.Cookie, error) {
	tr, ok := c.client.Transport.(*http.Transport)
	if !ok {
		return nil, errors.New("couldn't copy transport settings of the main HTTP to get cookies")
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create cookie jar for session renewal client")
	}

	sessionClient := &http.Client{
		Transport: &http.Transport{
			DialContext:         tr.DialContext,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: tr.TLSClientConfig.InsecureSkipVerify},
			IdleConnTimeout:     tr.IdleConnTimeout,
			MaxIdleConns:        tr.MaxIdleConns,
			MaxIdleConnsPerHost: tr.MaxIdleConnsPerHost,
			Proxy:               tr.Proxy,
		},
		CheckRedirect: redirectFunc,
		Jar:           jar,
	}

	var returnErr error

	for i := 0; i < getCookiesRepeatAttempts; i++ {
		cookiesReq, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
		if err != nil {
			returnErr = err
			continue
		}

		for header, value := range c.headers {
			cookiesReq.Header.Set(header, value)
		}
		cookiesReq.Host = c.hostHeader

		cookieResp, err := sessionClient.Do(cookiesReq)
		if err != nil {
			returnErr = err
			continue
		}
		cookieResp.Body.Close()

		return sessionClient.Jar.Cookies(cookiesReq.URL), nil
	}

	return nil, returnErr
}

func GetTargetURL(reqURL *url.URL) string {
	targetURL := *reqURL

	targetURL.Path = ""
	targetURL.RawPath = ""
	targetURL.ForceQuery = false
	targetURL.RawQuery = ""
	targetURL.Fragment = ""
	targetURL.RawFragment = ""

	return targetURL.String()
}
