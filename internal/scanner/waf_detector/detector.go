package waf_detector

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/wallarm/gotestwaf/internal/scanner/types"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/config"
	dns_cache "github.com/wallarm/gotestwaf/internal/dnscache"
	"github.com/wallarm/gotestwaf/internal/helpers"
	"github.com/wallarm/gotestwaf/internal/scanner/waf_detector/detectors"
	"github.com/wallarm/gotestwaf/pkg/dnscache"
)

const (
	xssPayload  = `<script>alert("XSS");</script>`
	sqliPayload = `UNION SELECT ALL FROM information_schema AND ' or SLEEP(5) or '`
	lfiPayload  = `../../../../etc/passwd`
	rcePayload  = `/bin/cat /etc/passwd; ping 127.0.0.1; curl google.com`
	xxePayload  = `<!ENTITY xxe SYSTEM "file:///etc/shadow">]><pwn>&hack;</pwn>`
)

type WAFDetector struct {
	clientSettings *ClientSettings
	headers        map[string]string
	hostHeader     string
	target         string
}

type ClientSettings struct {
	dnsResolver         *dnscache.Resolver
	insecureSkipVerify  bool
	idleConnTimeout     time.Duration
	maxIdleConns        int
	maxIdleConnsPerHost int
	proxyURL            *url.URL
}

func NewWAFDetector(logger *logrus.Logger, cfg *config.Config) (*WAFDetector, error) {
	clientSettings := &ClientSettings{
		insecureSkipVerify:  !cfg.TLSVerify,
		idleConnTimeout:     time.Duration(cfg.IdleConnTimeout) * time.Second,
		maxIdleConns:        cfg.MaxIdleConns,
		maxIdleConnsPerHost: cfg.MaxIdleConns,
	}

	dnsCache, err := dns_cache.NewDNSCache(logger)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create DNS cache")
	}

	clientSettings.dnsResolver = dnsCache

	if cfg.Proxy != "" {
		proxyURL, err := url.Parse(cfg.Proxy)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't parse proxy URL")
		}

		clientSettings.proxyURL = proxyURL
	}

	target, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't parse URL")
	}

	configuredHeaders := cfg.HTTPHeaders
	customHeader := strings.SplitN(cfg.AddHeader, ":", 2)
	if len(customHeader) > 1 {
		header := strings.TrimSpace(customHeader[0])
		value := strings.TrimSpace(customHeader[1])
		configuredHeaders[header] = value
	}

	return &WAFDetector{
		clientSettings: clientSettings,
		headers:        configuredHeaders,
		hostHeader:     configuredHeaders["Host"],
		target:         helpers.GetTargetURLStr(target),
	}, nil
}

func (w *WAFDetector) getHttpClient() (*http.Client, error) {
	tr := &http.Transport{
		DialContext:         dnscache.DialFunc(w.clientSettings.dnsResolver, nil),
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: w.clientSettings.insecureSkipVerify},
		IdleConnTimeout:     w.clientSettings.idleConnTimeout,
		MaxIdleConns:        w.clientSettings.maxIdleConns,
		MaxIdleConnsPerHost: w.clientSettings.maxIdleConns, // net.http hardcodes DefaultMaxIdleConnsPerHost to 2!
	}

	if w.clientSettings.proxyURL != nil {
		tr.Proxy = http.ProxyURL(w.clientSettings.proxyURL)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create cookie jar")
	}

	client := &http.Client{
		Transport: tr,
		Jar:       jar,
	}

	return client, nil
}

// doRequest sends HTTP-request without malicious payload to trigger WAF.
func (w *WAFDetector) doRequest(ctx context.Context) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.target, nil)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create request")
	}

	for header, value := range w.headers {
		req.Header.Set(header, value)
	}
	req.Host = w.hostHeader

	client, err := w.getHttpClient()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create HTTP client")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sent request")
	}

	return resp, nil
}

// doMaliciousRequest sends HTTP-request with malicious payload to trigger WAF.
func (w *WAFDetector) doMaliciousRequest(ctx context.Context) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.target, nil)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create request")
	}

	queryParams := req.URL.Query()
	queryParams.Add("a", xssPayload)
	queryParams.Add("b", sqliPayload)
	queryParams.Add("c", lfiPayload)
	queryParams.Add("d", rcePayload)
	queryParams.Add("e", xxePayload)

	req.URL.RawQuery = queryParams.Encode()

	for header, value := range w.headers {
		req.Header.Set(header, value)
	}
	req.Host = w.hostHeader

	client, err := w.getHttpClient()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create HTTP client")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sent request")
	}

	return resp, nil
}

// DetectWAF performs WAF identification. Returns WAF name and vendor after
// the first positive match.
func (w *WAFDetector) DetectWAF(ctx context.Context) (name, vendor string, checkFunc detectors.Check, err error) {
	resp, err := w.doRequest(ctx)
	if err != nil {
		return "", "", nil, errors.Wrap(err, "couldn't perform request without attack")
	}

	defer resp.Body.Close()

	respToAttack, err := w.doMaliciousRequest(ctx)
	if err != nil {
		return "", "", nil, errors.Wrap(err, "couldn't perform request with attack")
	}

	defer respToAttack.Body.Close()

	resps := &detectors.Responses{
		Resp:         &types.GoHTTPResponse{Resp: resp},
		RespToAttack: &types.GoHTTPResponse{Resp: respToAttack},
	}

	for _, d := range detectors.Detectors {
		if d.IsWAF(resps) {
			return d.WAFName, d.Vendor, d.Check, nil
		}
	}

	return "", "", nil, nil
}
