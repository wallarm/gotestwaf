package waf

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

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
