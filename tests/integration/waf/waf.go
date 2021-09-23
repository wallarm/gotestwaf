package waf

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

	ph "github.com/wallarm/gotestwaf/internal/payload/placeholder"
	"github.com/wallarm/gotestwaf/tests/integration/config"
)

type WAF struct {
	server *http.Server
}

func New(errChan chan<- error, casesMap *config.TestCasesMap) *WAF {
	mux := http.NewServeMux()
	mux.HandleFunc("/", createParseRequestFunc(errChan, casesMap))

	server := &http.Server{
		Addr:    config.Address,
		Handler: mux,
	}

	return &WAF{server: server}
}

func createParseRequestFunc(errChan chan<- error, casesMap *config.TestCasesMap) http.HandlerFunc {
	parseRequest := func(w http.ResponseWriter, r *http.Request) {
		testCase := r.Header.Get("X-GoTestWAF-Test")
		if testCase == "" {
			errChan <- errors.New("couldn't get X-GoTestWAF-Test header value")
		}

		params := strings.Split(testCase, "-")
		placeholder := params[0]
		encoder := params[1]

		var err error
		var placeholderValue string
		var value string

		switch placeholder {
		case "Header":
			placeholderValue, err = getPayloadFromHeader(r)
		case "RequestBody":
			placeholderValue, err = getPayloadFromRequestBody(r)
		case "SOAPBody":
			placeholderValue, err = getPayloadFromSOAPBody(r)
		case "JSONBody":
			placeholderValue, err = getPayloadFromJSONBody(r)
		case "URLParam":
			placeholderValue, err = getPayloadFromURLParam(r)
		case "URLPath":
			placeholderValue, err = getPayloadFromURLPath(r)
		}

		if err != nil {
			errChan <- fmt.Errorf("couldn't get encoded payload value: %v", err)
		}

		switch encoder {
		case "Base64":
			value, err = decodeBase64(placeholderValue)
		case "Base64Flat":
			value, err = decodeBase64(placeholderValue)
		case "JSUnicode":
			value, err = decodeJSUnicode(placeholderValue)
		case "URL":
			value, err = decodeURL(placeholderValue)
		case "Plain":
			value, err = decodePlain(placeholderValue)
		case "XMLEntity":
			value, err = decodeXMLEntity(placeholderValue)
		case "GRPC":
			value, err = decodeGRPC(placeholderValue)
		}

		if err != nil {
			errChan <- fmt.Errorf("couldn't decode payload: %v", err)
		}

		if matched, _ := regexp.MatchString("bypassed", value); matched {
			w.WriteHeader(http.StatusNonAuthoritativeInfo)
		} else if matched, _ = regexp.MatchString("blocked", value); matched {
			w.WriteHeader(http.StatusForbidden)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}

		testCase = fmt.Sprintf("%s-%s-%s", value, placeholder, encoder)
		if !casesMap.CheckTestCaseAvailability(testCase) {
			errChan <- fmt.Errorf("received unknown payload: %s", testCase)
		}
	}

	return parseRequest
}

func (w *WAF) Run() error {
	err := w.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (w *WAF) Shutdown() error {
	err := w.server.Shutdown(context.Background())
	return err
}

func getPayloadFromHeader(r *http.Request) (string, error) {
	re := fmt.Sprintf("X-[a-fA-F0-9]{%d}", ph.Seed*2)

	for header, values := range r.Header {
		if matched, _ := regexp.MatchString(re, header); matched {
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

	re := regexp.MustCompile(fmt.Sprintf("<ab[a-fA-F0-9]{%d}>.*</ab[a-fA-F0-9]{%[1]d}>", ph.Seed*2))
	match := re.FindAllString(string(body), -1)
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

	re := regexp.MustCompile(fmt.Sprintf("\"[a-fA-F0-9]{%d}\": \".*\"", ph.Seed*2))
	match := re.FindAllString(string(body), -1)
	if match == nil {
		return "", errors.New("couldn't get payload from JSON: payload not found")
	}

	return decodeJSUnicode(match[0][15 : len(match[0])-1])
}

func getPayloadFromURLParam(r *http.Request) (string, error) {
	re := fmt.Sprintf("[a-fA-F0-9]{%d}", ph.Seed*2)

	for key, values := range r.URL.Query() {
		if matched, _ := regexp.MatchString(re, key); matched {
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

func decodeBase64(payload string) (string, error) {
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	value, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("couldn't decode base64: %v", err)
	}

	return string(value), nil
}

func decodeJSUnicode(payload string) (string, error) {
	h := strings.ReplaceAll(payload, "\\u", "")

	utf16beStrBytes, err := hex.DecodeString(h)
	if err != nil {
		return "", fmt.Errorf("couldn't decode js unicode encoding: %v", err)
	}

	encoder := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder()
	value, _, err := transform.Bytes(encoder, utf16beStrBytes)
	if err != nil {
		return "", fmt.Errorf("couldn't decode js unicode encoding: %v", err)
	}

	return string(value), nil
}

func decodeURL(payload string) (string, error) {
	value, err := url.QueryUnescape(payload)
	if err != nil {
		return "", fmt.Errorf("couldn't decode URL encoding: %v", err)
	}
	return value, nil
}

func decodePlain(payload string) (string, error) {
	return payload, nil
}

func decodeXMLEntity(payload string) (string, error) {
	var res string
	b := bytes.NewBufferString(payload)
	if err := xml.NewDecoder(b).Decode(&res); err != nil {
		return "", fmt.Errorf("couldn't parse XML: %v", err)
	}
	return res, nil
}

func decodeGRPC(payload string) (string, error) {
	return payload, nil
}
