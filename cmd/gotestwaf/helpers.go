package main

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
)

const (
	httpProto    = "http"
	graphqlProto = httpProto
)

var (
	ErrInvalidScheme = errors.New("invalid URL scheme")
	ErrEmptyHost     = errors.New("empty host")
)

// validateURL validates the given URL and URL scheme.
func validateURL(rawURL string, protocol string) (*url.URL, error) {
	validURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(fmt.Sprintf("^%ss?$", protocol))

	if !re.MatchString(validURL.Scheme) {
		return nil, ErrInvalidScheme
	}

	if validURL.Host == "" {
		return nil, ErrEmptyHost
	}

	return validURL, nil
}

// checkOrCraftProtocolURL creates a URL from validHttpURL if the rawURL is empty
// or validates the rawURL.
func checkOrCraftProtocolURL(rawURL string, validHttpURL string, protocol string) (*url.URL, error) {
	if rawURL != "" {
		validURL, err := validateURL(rawURL, protocol)
		if err != nil {
			return nil, err
		}

		return validURL, nil
	}

	validURL, err := validateURL(validHttpURL, httpProto)
	if err != nil {
		return nil, err
	}

	scheme := protocol

	if validURL.Scheme == "https" {
		scheme += "s"
	}

	validURL.Scheme = scheme
	validURL.Path = ""

	return validURL, nil
}

func validateHttpClient(httpClient string) error {
	if _, ok := httpClientsSet[httpClient]; !ok {
		return fmt.Errorf("invalid HTTP client: %s", httpClient)
	}

	return nil
}

func validateLogFormat(logFormat string) error {
	if _, ok := logFormatsSet[logFormat]; !ok {
		return fmt.Errorf("invalid log format: %s", logFormat)
	}

	return nil
}
