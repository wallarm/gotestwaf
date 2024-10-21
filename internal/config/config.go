package config

import (
	"github.com/wallarm/gotestwaf/internal/scanner/waf_detector/detectors"
)

type Config struct {
	// Target settings
	URL         string `mapstructure:"url"`
	GRPCPort    uint16 `mapstructure:"grpcPort"`
	GraphQLURL  string `mapstructure:"graphqlURL"`
	OpenAPIFile string `mapstructure:"openapiFile"`

	// Test cases settings
	TestCase      string `mapstructure:"testCase"`
	TestCasesPath string `mapstructure:"testCasesPath"`
	TestSet       string `mapstructure:"testSet"`

	// HTTP client settings
	HTTPClient     string `mapstructure:"httpClient"`
	TLSVerify      bool   `mapstructure:"tlsVerify"`
	Proxy          string `mapstructure:"proxy"`
	AddHeader      string `mapstructure:"addHeader"`
	AddDebugHeader bool   `mapstructure:"addDebugHeader"`

	// GoHTTP client only settings
	MaxIdleConns    int  `mapstructure:"maxIdleConns"`
	MaxRedirects    int  `mapstructure:"maxRedirects"`
	IdleConnTimeout int  `mapstructure:"idleConnTimeout"`
	FollowCookies   bool `mapstructure:"followCookies"`
	RenewSession    bool `mapstructure:"renewSession"`

	// Performance settings
	Workers     int `mapstructure:"workers"`
	RandomDelay int `mapstructure:"randomDelay"`
	SendDelay   int `mapstructure:"sendDelay"`

	// Analysis settings
	SkipWAFBlockCheck     bool   `mapstructure:"skipWAFBlockCheck"`
	SkipWAFIdentification bool   `mapstructure:"skipWAFIdentification"`
	BlockStatusCodes      []int  `mapstructure:"blockStatusCodes"`
	PassStatusCodes       []int  `mapstructure:"passStatusCodes"`
	BlockRegex            string `mapstructure:"blockRegex"`
	PassRegex             string `mapstructure:"passRegex"`
	NonBlockedAsPassed    bool   `mapstructure:"nonBlockedAsPassed"`
	IgnoreUnresolved      bool   `mapstructure:"ignoreUnresolved"`
	BlockConnReset        bool   `mapstructure:"blockConnReset"`

	// Report settings
	WAFName          string   `mapstructure:"wafName"`
	IncludePayloads  bool     `mapstructure:"includePayloads"`
	ReportPath       string   `mapstructure:"reportPath"`
	ReportName       string   `mapstructure:"reportName"`
	ReportFormat     []string `mapstructure:"reportFormat"`
	NoEmailReport    bool     `mapstructure:"noEmailReport"`
	Email            string   `mapstructure:"email"`
	HideArgsInReport bool     `mapstructure:"hideArgsInReport"`

	// config.yaml
	HTTPHeaders map[string]string `mapstructure:"headers"`

	// Other settings
	LogLevel string `mapstructure:"logLevel"`

	CheckBlockFunc detectors.Check

	Args []string
}
