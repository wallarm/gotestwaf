package waf

import (
	"errors"
	"fmt"
	"io"
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

func getPayloadFromUAHeader(r *http.Request) (string, error) {
	if header := r.UserAgent(); header != "" {
		return header, nil
	}

	return "", errors.New("couldn't get payload from UA header: required header not found")
}

func getPayloadFromHeader(r *http.Request) (string, error) {
	for header, values := range r.Header {
		if matched := headerRegexp.MatchString(header); matched {
			return values[0], nil
		}
	}

	return "", errors.New("couldn't get payload from header: required header not found")
}

func getPayloadFromHTMLForm(r *http.Request) (string, error) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", fmt.Errorf("couldn't get payload from form body: %v", err)
	}

	payload := string(body)

	match := urlParamRegexp.MatchString(payload)
	if !match {
		return "", errors.New("couldn't get payload from form body: payload not found")
	}

	return payload[ph.Seed*2+1:], nil
}

func getPayloadFromHTMLMultipartForm(r *http.Request) (string, error) {
	err := r.ParseMultipartForm(1 << 10)
	if err != nil {
		return "", fmt.Errorf("couldn't parse multipart form: %v", err)
	}

	for paramName, values := range r.MultipartForm.Value {
		if matched := urlParamRegexp.MatchString(paramName); matched {
			return values[0], nil
		}
	}

	return "", errors.New("couldn't get payload from multipart form body")
}

func getPayloadFromJSONBody(r *http.Request) (string, error) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", fmt.Errorf("couldn't get payload from JSON body: %v", err)
	}
	return string(body), nil
}

func getPayloadFromJSONRequest(r *http.Request) (string, error) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", fmt.Errorf("couldn't read request body: %v", err)
	}

	match := jsonBodyRegexp.FindAllString(string(body), -1)
	if match == nil {
		return "", errors.New("couldn't get payload from JSON: payload not found")
	}

	return decodeJSUnicode(match[0][15 : len(match[0])-1])
}

func getPayloadFromRequestBody(r *http.Request) (string, error) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", fmt.Errorf("couldn't get payload from request body: %v", err)
	}
	return string(body), nil
}

func getPayloadFromSOAPBody(r *http.Request) (string, error) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", fmt.Errorf("couldn't read request body: %v", err)
	}

	match := soapBodyRegexp.FindAllString(string(body), -1)
	if match == nil {
		return "", errors.New("couldn't get payload from SOAP body: payload not found")
	}

	return decodeXMLEntity(match[0][14 : len(match[0])-15])
}

func getPayloadFromURLParam(r *http.Request) (string, error) {
	for key, values := range r.URL.Query() {
		if matched := urlParamRegexp.MatchString(key); matched {
			return values[0], nil
		}
	}

	return "", errors.New("couldn't get payload from URL parameters: required parameter not found")
}

func getPayloadFromURLPath(r *http.Request) (payload string, err error) {
	defer func() {
		if recoveryMessage := recover(); recoveryMessage != nil {
			payload = ""
			err = fmt.Errorf("couldn't get payload from URL path: %s", recoveryMessage)
		}
	}()

	payload = r.URL.Path[1:]

	return payload, nil
}

func getPayloadFromXMLBody(r *http.Request) (string, error) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", fmt.Errorf("couldn't get payload from XML body: %v", err)
	}
	return string(body), nil
}
