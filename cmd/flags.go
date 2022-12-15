package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/wallarm/gotestwaf/internal/config"
	"github.com/wallarm/gotestwaf/internal/helpers"
	"github.com/wallarm/gotestwaf/internal/version"
)

const (
	maxReportFilenameLength = 249 // 255 (max length) - 5 (".html") - 1 (to be sure)

	defaultReportPath    = "reports"
	defaultReportName    = "waf-evaluation-report-2006-January-02-15-04-05"
	defaultTestCasesPath = "testcases"
	defaultConfigPath    = "config.yaml"

	wafName = "generic"

	textLogFormat = "text"
	jsonLogFormat = "json"

	cliDescription = `GoTestWAF is a tool for API and OWASP attack simulation that supports a
wide range of API protocols including REST, GraphQL, gRPC, WebSockets,
SOAP, XMLRPC, and others.
Homepage: https://github.com/wallarm/gotestwaf

Usage: %s [OPTIONS] --url <URL>

Options:
`
)

var (
	configPath string
	quiet      bool
	logLevel   logrus.Level
	logFormat  string

	isIncludePayloadsFlagUsed bool
)

var usage = func() {
	flag.CommandLine.SetOutput(os.Stdout)
	usage := cliDescription
	fmt.Fprintf(os.Stdout, usage, os.Args[0])
	flag.PrintDefaults()
}

// parseFlags parses all GoTestWAF CLI flags
func parseFlags() (args []string, err error) {
	reportPath := filepath.Join(".", defaultReportPath)
	testCasesPath := filepath.Join(".", defaultTestCasesPath)

	flag.Usage = usage

	flag.StringVar(&configPath, "configPath", defaultConfigPath, "Path to the config file")
	flag.BoolVar(&quiet, "quiet", false, "If true, disable verbose logging")
	logLvl := flag.String("logLevel", "info", "Logging level: panic, fatal, error, warn, info, debug, trace")
	flag.StringVar(&logFormat, "logFormat", textLogFormat, "Set logging format: text, json")

	urlParam := flag.String("url", "", "URL to check")
	wsURL := flag.String("wsURL", "", "WebSocket URL to check")
	flag.Uint16("grpcPort", 0, "gRPC port to check")
	flag.String("proxy", "", "Proxy URL to use")
	flag.Bool("tlsVerify", false, "If true, the received TLS certificate will be verified")
	flag.Int("maxIdleConns", 2, "The maximum number of keep-alive connections")
	flag.Int("maxRedirects", 50, "The maximum number of handling redirects")
	flag.Int("idleConnTimeout", 2, "The maximum amount of time a keep-alive connection will live")
	flag.Bool("followCookies", false, "If true, use cookies sent by the server. May work only with --maxIdleConns=1")
	flag.Bool("renewSession", false, "Renew cookies before each test. Should be used with --followCookies flag")
	flag.Bool("skipWAFIdentification", false, "Skip WAF identification")
	flag.IntSlice("blockStatusCodes", []int{403}, "HTTP status code that WAF uses while blocking requests")
	flag.IntSlice("passStatusCodes", []int{200, 404}, "HTTP response status code that WAF uses while passing requests")
	blockRegex := flag.String("blockRegex", "",
		"Regex to detect a blocking page with the same HTTP response status code as a not blocked request")
	passRegex := flag.String("passRegex", "",
		"Regex to a detect normal (not blocked) web page with the same HTTP status code as a blocked request")
	flag.Bool("nonBlockedAsPassed", false,
		"If true, count requests that weren't blocked as passed. If false, requests that don't satisfy to PassStatusCodes/PassRegExp as blocked")
	flag.Int("workers", 5, "The number of workers to scan")
	flag.Int("sendDelay", 400, "Delay in ms between requests")
	flag.Int("randomDelay", 400, "Random delay in ms in addition to the delay between requests")
	flag.String("testCase", "", "If set then only this test case will be run")
	flag.String("testSet", "", "If set then only this test set's cases will be run")
	flag.String("reportPath", reportPath, "A directory to store reports")
	reportName := flag.String("reportName", defaultReportName, "Report file name. Supports `time' package template format")
	flag.String("reportFormat", "pdf", "Export report to one of the following formats: none, pdf, html, json")
	flag.Bool("includePayloads", false, "If true, payloads will be included in HTML/PDF report")
	noEmailReport := flag.Bool("noEmailReport", false, "Save report locally")
	email := flag.String("email", "", "E-mail to which the report will be sent")
	flag.String("testCasesPath", testCasesPath, "Path to a folder with test cases")
	flag.String("wafName", wafName, "Name of the WAF product")
	flag.Bool("ignoreUnresolved", false, "If true, unresolved test cases will be considered as bypassed (affect score and results)")
	flag.Bool("blockConnReset", false, "If true, connection resets will be considered as block")
	flag.Bool("skipWAFBlockCheck", false, "If true, WAF detection tests will be skipped")
	flag.String("addHeader", "", "An HTTP header to add to requests")
	flag.Bool("addDebugHeader", false, "Add header with a hash of the test information in each request")
	flag.String("openapiFile", "", "Path to openAPI file")
	showVersion := flag.Bool("version", false, "Show GoTestWAF version and exit")
	flag.Parse()

	if len(os.Args) == 1 {
		usage()
		os.Exit(0)
	}

	// show version and exit
	if *showVersion == true {
		fmt.Fprintf(os.Stderr, "GoTestWAF %s\n", version.Version)
		os.Exit(0)
	}

	// url flag must be set
	if *urlParam == "" {
		return nil, errors.New("--url flag is not set")
	}

	if *noEmailReport == false && *email != "" {
		*email, err = helpers.ValidateEmail(*email)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't validate email")
		}
	}

	logrusLogLvl, err := logrus.ParseLevel(*logLvl)
	if err != nil {
		return nil, err
	}
	logLevel = logrusLogLvl

	if logFormat != textLogFormat && logFormat != jsonLogFormat {
		return nil, fmt.Errorf("unknown logging format: %s", logFormat)
	}

	validURL, err := url.Parse(*urlParam)
	if err != nil ||
		(validURL.Scheme != "http" && validURL.Scheme != "https") ||
		validURL.Host == "" {
		return nil, errors.New("URL is not valid")
	}

	*urlParam = validURL.String()

	// format WebSocket URL from given HTTP URL
	if *wsURL == "" {
		wsScheme := "ws"
		if validURL.Scheme == "https" {
			wsScheme = "wss"
		}
		validURL.Scheme = wsScheme
		validURL.Path = ""

		*wsURL = validURL.String()
	}

	if *blockRegex != "" {
		_, err = regexp.Compile(*blockRegex)
		if err != nil {
			return nil, errors.Wrap(err, "bad regexp")
		}
	}

	if *passRegex != "" {
		_, err = regexp.Compile(*passRegex)
		if err != nil {
			return nil, errors.Wrap(err, "bad regexp")
		}
	}

	_, reportFileName := filepath.Split(*reportName)
	if len(reportFileName) > maxReportFilenameLength {
		return nil, errors.New("report filename too long")
	}

	checkUsedFlags()

	args, err = normalizeArgs()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't normalize args")
	}

	return args, nil
}

func checkUsedFlags() {
	fn := func(f *flag.Flag) {
		if f.Name == "includePayloads" {
			isIncludePayloadsFlagUsed = f.Changed
		}
	}

	flag.Visit(fn)
}

// normalizeArgs returns string with used CLI args in a unified from.
func normalizeArgs() ([]string, error) {
	// disable lexicographical order
	flag.CommandLine.SortFlags = false

	var (
		args []string
		err  error
	)

	fn := func(f *flag.Flag) {
		// skip if flag wasn't changed
		if !f.Changed {
			return
		}

		var (
			value string
			arg   string
		)

		// all types listed in parseFlags function
		argType := f.Value.Type()
		switch argType {
		case "string":
			value = strings.TrimSpace(f.Value.String())

			if strings.Contains(value, " ") {
				value = `"` + value + `"`
			}

			arg = fmt.Sprintf("--%s=%s", f.Name, value)

		case "bool":
			arg = fmt.Sprintf("--%s", f.Name)

		case "int", "uint16":
			value = f.Value.String()
			arg = fmt.Sprintf("--%s=%s", f.Name, value)

		case "intSlice":
			// remove square brackets: [200,404] -> 200,404
			value = strings.Trim(f.Value.String(), "[]")
			arg = fmt.Sprintf("--%s=%s", f.Name, value)

		default:
			err = multierror.Append(err, fmt.Errorf("unknown CLI argument type: %s", argType))
		}

		args = append(args, arg)
	}

	// get all changed flags
	flag.Visit(fn)

	if err != nil {
		return nil, err
	}

	return args, nil
}

// loadConfig loads the specified config file and merges it with the parameters passed via CLI
func loadConfig() (cfg *config.Config, err error) {
	err = viper.BindPFlags(flag.CommandLine)
	if err != nil {
		return nil, err
	}
	viper.AddConfigPath(".")
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}
	err = viper.Unmarshal(&cfg)
	return
}
