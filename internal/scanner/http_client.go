package scanner

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/config"
	"github.com/wallarm/gotestwaf/internal/payload/encoder"
	"github.com/wallarm/gotestwaf/internal/payload/placeholder"
)

type HTTPClient struct {
	client        *http.Client
	cookies       []*http.Cookie
	headers       map[string]string
	followCookies bool
}

func NewHTTPClient(cfg *config.Config) (*HTTPClient, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !cfg.TLSVerify},
		IdleConnTimeout: time.Duration(cfg.IdleConnTimeout) * time.Second,
		MaxIdleConns:    cfg.MaxIdleConns,
	}

	if cfg.Proxy != "" {
		proxyURL, _ := url.Parse(cfg.Proxy)
		tr.Proxy = http.ProxyURL(proxyURL)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	cl := &http.Client{
		Transport: tr,
		CheckRedirect: func() func(req *http.Request, via []*http.Request) error {
			redirects := 0
			return func(req *http.Request, via []*http.Request) error {
				if redirects > cfg.MaxRedirects {
					return errors.New("max redirect number exceeded")
				}
				redirects++
				return nil
			}
		}(),
		Jar: jar,
	}

	configuredHeaders := cfg.HTTPHeaders
	customHeader := strings.Split(cfg.AddHeader, ":")
	if len(customHeader) > 1 {
		configuredHeaders[customHeader[0]] = strings.TrimPrefix(cfg.AddHeader, customHeader[0]+":")
	}

	return &HTTPClient{
		client:        cl,
		cookies:       cfg.Cookies,
		headers:       configuredHeaders,
		followCookies: cfg.FollowCookies,
	}, nil
}

func (c *HTTPClient) Send(
	ctx context.Context,
	targetURL, placeholderName, encoderName, payload string,
	testHeaderValue string,
) (body []byte, statusCode int, err error) {
	encodedPayload, err := encoder.Apply(encoderName, payload)
	if err != nil {
		return nil, 0, errors.Wrap(err, "encoding payload")
	}

	req, err := placeholder.Apply(targetURL, placeholderName, encodedPayload)
	if err != nil {
		return nil, 0, errors.Wrap(err, "apply placeholder")
	}

	req = req.WithContext(ctx)

	for header, value := range c.headers {
		req.Header.Set(header, value)
	}

	if testHeaderValue != "" {
		req.Header.Set("X-GoTestWAF-Test", testHeaderValue)
	}

	if len(c.cookies) > 0 && c.followCookies {
		c.client.Jar.SetCookies(req.URL, c.cookies)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, errors.Wrap(err, "sending http request")
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, errors.Wrap(err, "reading response body")
	}
	statusCode = resp.StatusCode

	if len(resp.Cookies()) > 0 {
		c.cookies = append(c.cookies, resp.Cookies()...)
	}

	return body, statusCode, nil
}
