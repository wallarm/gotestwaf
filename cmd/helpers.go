package main

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
)

const (
	httpProto    = "http"
	wsProto      = "ws"
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
