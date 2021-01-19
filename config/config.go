package config

import "net/http"

type Config struct {
	Cookies            []*http.Cookie
	URL                string            `mapstructure:"url"`
	ConfigPath         string            `mapstructure:"configPath"`
	HTTPHeaders        map[string]string `mapstructure:"headers"`
	Proxy              string            `mapstructure:"proxy"`
	TLSVerify          bool              `mapstructure:"tlsverify"`
	MaxIdleConns       int               `mapstructure:"maxIdleConns"`
	IdleConnTimeout    int               `mapstructure:"idleConnTimeout"`
	TestCasesPath      string            `mapstructure:"testCasesPath"`
	BlockStatusCode    int               `mapstructure:"blockStatusCode"`
	BlockRegExp        string            `mapstructure:"blockRegExp"`
	PassStatusCode     int               `mapstructure:"passStatusCode"`
	PassRegExp         string            `mapstructure:"passRegExp"`
	ReportDir          string            `mapstructure:"reportDir"`
	NonBlockedAsPassed bool              `mapstructure:"nonBlockedAsPassed"`
	FollowCookies      bool              `mapstructure:"followCookies"`
	MaxRedirects       int               `mapstructure:"maxRedirects"`
	SendDelay          int               `mapstructure:"sendDelay"`
	RandomDelay        int               `mapstructure:"randomDelay"`
}
