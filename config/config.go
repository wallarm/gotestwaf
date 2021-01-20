package config

import "net/http"

type Config struct {
	Cookies            []*http.Cookie
	URL                string            `mapstructure:"url"`
	HTTPHeaders        map[string]string `mapstructure:"headers"`
	TLSVerify          bool              `mapstructure:"tlsverify"`
	Proxy              string            `mapstructure:"proxy"`
	MaxIdleConns       int               `mapstructure:"maxIdleConns"`
	MaxRedirects       int               `mapstructure:"maxRedirects"`
	IdleConnTimeout    int               `mapstructure:"idleConnTimeout"`
	FollowCookies      bool              `mapstructure:"followCookies"`
	BlockStatusCode    int               `mapstructure:"blockStatusCode"`
	PassStatusCode     int               `mapstructure:"passStatusCode"`
	BlockRegex         string            `mapstructure:"blockRegex"`
	PassRegex          string            `mapstructure:"passRegex"`
	NonBlockedAsPassed bool              `mapstructure:"nonBlockedAsPassed"`
	RandomDelay        int               `mapstructure:"randomDelay"`
	SendDelay          int               `mapstructure:"sendDelay"`
	TestCasesPath      string            `mapstructure:"testCasesPath"`
	ReportDir          string            `mapstructure:"reportDir"`
}
