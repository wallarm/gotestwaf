package config

import (
	"net/http"
)

type Config struct {
	Cookies               []*http.Cookie
	URL                   string            `conf:"default:http://127.0.0.1:8080,help:URL with a WAF to check."`
	YAMLConfigPath        string            `conf:"default:config.yaml,help:YAML config file."`
	HTTPHeaders           map[string]string `yaml:"headers"`
	Proxy                 string            `conf:"help:Proxy to use" yaml:"proxy"`
	TLSVerify             bool              `conf:"default:false,help:Verify or not TLS certificate" yaml:"tlsverify"`
	MaxIdleConnections    int               `conf:"default:2,help:Number of concurrent threads sending requests." yaml:"threads"`
	IdleConnectionTimeout int               `conf:"default:100" yaml:"threadTimeout"`
	TestCasesFolder       string            `conf:"default:./testcases,help:Folder with test cases." yaml:"testCasesFolder"`
	BlockStatusCode       int               `conf:"default:403,help:HTTP response status code that WAF use while blocking requests." yaml:"blockStatusCode"`
	BlockRegExp           string            `conf:"help:Regex to detect blocking page with the same HTTP response status code as not blocked request." yaml:"blockRegExp"`
	PassStatusCode        int               `conf:"default:200,help:HTTP response status code that WAF use while passing requests." yaml:"passStatusCode"`
	PassRegExp            string            `conf:"help:Regex to detect normal (not blocked) web-page with the same HTTP response status code as blocked request." yaml:"passRegExp"`
	ReportDir            string            `conf:"default:/tmp/gotestwaf,help:PDF report filename used to export results." yaml:"reportFile"`
	NonBlockedAsPassed    bool              `conf:"default:true,help:Count all the requests that were not blocked as passed. Otherwise, count all of them that doens't satisfy PassStatuscode/PassRegExp as blocked (by default)" yaml:"nonBlockedAsPassed"`
	FollowCookies         bool `conf:"default:false,help:Use cookies sent by server. May work only for --threads=1" yaml:"followCookies"`
	MaxRedirects          int  `conf:"default:50,help:Maximum amount of redirects per request that GoTestWAF will follow until the hard stop" yaml:"maxRedirects"`
	SendingDelay          int  `conf:"default:500,help:Delay between sending requests inside threads, milliseconds" yaml:"sendingDelay"`
	RandomDelay           int  `conf:"default:500,help:Random delay, in addition to --sending_delay between requests inside threads, milliseconds" yaml:"randomDelay"`
}
