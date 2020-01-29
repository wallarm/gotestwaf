package payload

import (
	"crypto/tls"
	"gotestwaf/config"
	"gotestwaf/payload/encoder"
	"gotestwaf/payload/placeholder"
	"log"
	"net/http"
	"net/url"
	"time"
)

func Send(config config.Config, targetUrl string, placeholderName string, encoderName string, payload string) *http.Response {
	encodedPayload, _ := encoder.Apply(encoderName, payload)
	var req = placeholder.Apply(targetUrl, placeholderName, encodedPayload)
	//TODO: move certificates check into the config settings
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: config.CertificateCheck},
		IdleConnTimeout: time.Duration(config.IddleConnectionTimeout) * time.Second,
		MaxIdleConns:    config.MaxIddleConnections,
	}
	if config.Proxy != "" {
		proxyUrl, _ := url.Parse(config.Proxy)
		tr = &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
		}
	}
	for header, value := range config.Headers {
		req.Header.Set(header, value)
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	return resp
}
