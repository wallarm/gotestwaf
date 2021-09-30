package waf

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	ph "github.com/wallarm/gotestwaf/internal/payload/placeholder"
)

var (
	headerRegexp   = regexp.MustCompile(fmt.Sprintf("X-[a-fA-F0-9]{%d}", ph.Seed*2))
	soapBodyRegexp = regexp.MustCompile(fmt.Sprintf("<ab[a-fA-F0-9]{%d}>.*</ab[a-fA-F0-9]{%[1]d}>", ph.Seed*2))
	jsonBodyRegexp = regexp.MustCompile(fmt.Sprintf("\"[a-fA-F0-9]{%d}\": \".*\"", ph.Seed*2))
	urlParamRegexp = regexp.MustCompile(fmt.Sprintf("[a-fA-F0-9]{%d}", ph.Seed*2))
)

func getPayloadFromHeader(r *http.Request) (string, error) {
	for header, values := range r.Header {
		if matched := headerRegexp.MatchString(header); matched {
			return values[0], nil
		}
	}

	return "", errors.New("couldn't get payload from header: required header not found")
}

func getPayloadFromRequestBody(r *http.Request) (string, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return "", fmt.Errorf("couldn't get payload from request body: %v", err)
	}
	return string(body), nil
}

func getPayloadFromSOAPBody(r *http.Request) (string, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return "", fmt.Errorf("couldn't read request body: %v", err)
	}

	match := soapBodyRegexp.FindAllString(string(body), -1)
	if match == nil {
		return "", errors.New("couldn't get payload from SOAP body: payload not found")
	}

	return decodeXMLEntity(match[0][14 : len(match[0])-15])
}

func getPayloadFromJSONBody(r *http.Request) (string, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return "", fmt.Errorf("couldn't read request body: %v", err)
	}

	match := jsonBodyRegexp.FindAllString(string(body), -1)
	if match == nil {
		return "", errors.New("couldn't get payload from JSON: payload not found")
	}

	return decodeJSUnicode(match[0][15 : len(match[0])-1])
}

func getPayloadFromURLParam(r *http.Request) (string, error) {
	for key, values := range r.URL.Query() {
		if matched := urlParamRegexp.MatchString(key); matched {
			return values[0], nil
		}
	}

	return "", errors.New("couldn't get payload from URL parameters: required parameter not found")
}

func getPayloadFromURLPath(r *http.Request) (string, error) {
	payload := r.URL.Path[1 : len(r.URL.Path)-1]
	if recoveryMessage := recover(); recoveryMessage != nil {
		return "", fmt.Errorf("couldn't get payload from URL path: %s", recoveryMessage)
	}

	return payload, nil
}
