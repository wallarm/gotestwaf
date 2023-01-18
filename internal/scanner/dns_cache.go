package scanner

import (
	"time"

	"go.mercari.io/go-dnscache"
	"go.uber.org/zap"
)

const (
	dnsRefreshTime   = 1 * time.Minute
	dnsLookupTimeout = 10 * time.Second
)

func NewDNSCache() (*dnscache.Resolver, error) {
	dnsResolver, err := dnscache.New(dnsRefreshTime, dnsLookupTimeout, zap.NewNop())
	if err != nil {
		return nil, err
	}

	return dnsResolver, nil
}
