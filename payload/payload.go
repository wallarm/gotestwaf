package payload

import (
	"crypto/tls"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/wallarm/gotestwaf/config"
	"github.com/wallarm/gotestwaf/payload/encoder"
	"github.com/wallarm/gotestwaf/payload/placeholder"
)

func Send(cfg *config.Config, targetURL, placeholderName, encoderName, payload string) *http.Response {
	encodedPayload, err := encoder.Apply(encoderName, payload)
	if err != nil {
		log.Fatal("sending error:", err)
	}
	req := placeholder.Apply(targetURL, placeholderName, encodedPayload)
	//TODO: move certificates check into the cfg settings

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
	for header, value := range cfg.HTTPHeaders {
		req.Header.Set(header, value)
	}
	client := &http.Client{
		Transport: tr,
		// CheckRedirect: func(req *http.Request, via []*http.Request) error {
		//	return http.ErrUseLastResponse
		//}
		CheckRedirect: func() func(req *http.Request, via []*http.Request) error {
			redirects := 0
			return func(req *http.Request, via []*http.Request) error {
				if redirects > cfg.MaxRedirects {
					log.Fatal("Max redirect exceeded. Use --max_redirects to increase the limit")
				}
				redirects++
				return nil
			}
		}(),
	}
	if len(cfg.Cookies) > 0 && cfg.FollowCookies {
		log.Println(cfg.Cookies)
		client.Jar.SetCookies(req.URL, cfg.Cookies)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if len(resp.Cookies()) > 0 {
		cfg.Cookies = resp.Cookies()
	}

	return resp
}
