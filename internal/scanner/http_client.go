package scanner

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/wallarm/gotestwaf/internal/data/config"
	"github.com/wallarm/gotestwaf/internal/payload/encoder"
	"github.com/wallarm/gotestwaf/internal/payload/placeholder"
)

type HTTPClient struct {
	client        *http.Client
	cookies       []*http.Cookie
	headers       map[string]string
	followCookies bool
}

func NewHTTPClient(cfg *config.Config) *HTTPClient {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !cfg.TLSVerify},
		IdleConnTimeout: time.Duration(cfg.IdleConnTimeout) * time.Second,
		MaxIdleConns:    cfg.MaxIdleConns,
	}

	if cfg.Proxy != "" {
		proxyURL, _ := url.Parse(cfg.Proxy)
		tr = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
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
	}

	return &HTTPClient{
		client:        cl,
		cookies:       cfg.Cookies,
		headers:       cfg.HTTPHeaders,
		followCookies: cfg.FollowCookies,
	}
}

func (c *HTTPClient) Send(
	ctx context.Context, targetURL, placeholderName, encoderName, payload string) (
	body []byte, statusCode int, err error) {
	encodedPayload, err := encoder.Apply(encoderName, payload)
	if err != nil {
		return nil, 0, errors.Wrap(err, "encoding payload")
	}

	req := placeholder.Apply(targetURL, placeholderName, encodedPayload)
	req = req.WithContext(ctx)

	for header, value := range c.headers {
		req.Header.Set(header, value)
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
