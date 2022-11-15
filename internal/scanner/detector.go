package scanner

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/config"
	"github.com/wallarm/gotestwaf/internal/scanner/detectors"
)

const (
	xssPayload  = `<script>alert("XSS");</script>`
	sqliPayload = `UNION SELECT ALL FROM information_schema AND ' or SLEEP(5) or '`
	lfiPayload  = `../../../../etc/passwd`
	rcePayload  = `/bin/cat /etc/passwd; ping 127.0.0.1; curl google.com`
	xxePayload  = `<!ENTITY xxe SYSTEM "file:///etc/shadow">]><pwn>&hack;</pwn>`
)

type WAFDetector struct {
	client *http.Client
	target string
}

func NewDetector(cfg *config.Config) (*WAFDetector, error) {
	tr := &http.Transport{
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

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create cookie jar")
	}

	client := &http.Client{
		Transport: tr,
		Jar:       jar,
	}

	target, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't parse URL")
	}

	return &WAFDetector{
		client: client,
		target: GetTargetURL(target),
	}, nil
}

// doRequest sends HTTP-request with malicious payload to trigger WAF.
func (w *WAFDetector) doRequest(ctx context.Context) (*http.Response, error) {
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

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sent request")
	}

	return resp, nil
}

// DetectWAF performs WAF identification. Returns WAF name and vendor after
// the first positive match.
func (w *WAFDetector) DetectWAF(ctx context.Context) (name, vendor string, err error) {
	resp, err := w.doRequest(ctx)
	defer resp.Body.Close()
	if err != nil {
		return "", "", errors.Wrap(err, "couldn't identify WAF")
	}

	for _, d := range detectors.Detectors {
		if d.IsWAF(resp) {
			return d.WAFName, d.Vendor, nil
		}
	}

	return "", "", nil
}
