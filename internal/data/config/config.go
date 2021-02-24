package config

import "net/http"

type Config struct {
	Cookies            []*http.Cookie
	URL                string            `mapstructure:"url"`
	WebSocketURL       string            `mapstructure:"wsURL"`
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
	Workers            int               `mapstructure:"workers"`
	RandomDelay        int               `mapstructure:"randomDelay"`
	SendDelay          int               `mapstructure:"sendDelay"`
	ReportDir          string            `mapstructure:"reportDir"`
	TestCase           string            `mapstructure:"testCase"`
	TestCasesPath      string            `mapstructure:"testCasesPath"`
	TestSet            string            `mapstructure:"testSet"`
	WAFName            string            `mapstructure:"wafName"`
}
