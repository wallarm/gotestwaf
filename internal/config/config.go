package config

type Config struct {
	URL                   string            `mapstructure:"url"`
	WebSocketURL          string            `mapstructure:"wsURL"`
	GRPCPort              uint16            `mapstructure:"grpcPort"`
	HTTPHeaders           map[string]string `mapstructure:"headers"`
	TLSVerify             bool              `mapstructure:"tlsVerify"`
	Proxy                 string            `mapstructure:"proxy"`
	MaxIdleConns          int               `mapstructure:"maxIdleConns"`
	MaxRedirects          int               `mapstructure:"maxRedirects"`
	IdleConnTimeout       int               `mapstructure:"idleConnTimeout"`
	FollowCookies         bool              `mapstructure:"followCookies"`
	RenewSession          bool              `mapstructure:"renewSession"`
	SkipWAFIdentification bool              `mapstructure:"skipWAFIdentification"`
	BlockStatusCodes      []int             `mapstructure:"blockStatusCodes"`
	PassStatusCodes       []int             `mapstructure:"passStatusCodes"`
	BlockRegex            string            `mapstructure:"blockRegex"`
	PassRegex             string            `mapstructure:"passRegex"`
	NonBlockedAsPassed    bool              `mapstructure:"nonBlockedAsPassed"`
	Workers               int               `mapstructure:"workers"`
	RandomDelay           int               `mapstructure:"randomDelay"`
	SendDelay             int               `mapstructure:"sendDelay"`
	ReportPath            string            `mapstructure:"reportPath"`
	ReportName            string            `mapstructure:"reportName"`
	ReportFormat          string            `mapstructure:"reportFormat"`
	IncludePayloads       bool              `mapstructure:"includePayloads"`
	NoEmailReport         bool              `mapstructure:"noEmailReport"`
	Email                 string            `mapstructure:"email"`
	TestCase              string            `mapstructure:"testCase"`
	TestCasesPath         string            `mapstructure:"testCasesPath"`
	TestSet               string            `mapstructure:"testSet"`
	WAFName               string            `mapstructure:"wafName"`
	IgnoreUnresolved      bool              `mapstructure:"ignoreUnresolved"`
	BlockConnReset        bool              `mapstructure:"blockConnReset"`
	SkipWAFBlockCheck     bool              `mapstructure:"skipWAFBlockCheck"`
	AddHeader             string            `mapstructure:"addHeader"`
	AddDebugHeader        bool              `mapstructure:"addDebugHeader"`
	OpenAPIFile           string            `mapstructure:"openapiFile"`
}
