package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/wallarm/gotestwaf/internal/data/config"
	"github.com/wallarm/gotestwaf/internal/data/test"
	"github.com/wallarm/gotestwaf/internal/scanner"
)

const (
	defaultReportPath    = "reports"
	defaultTestCasesPath = "testcases"
	defaultConfigPath    = "config.yaml"

	wafName       = "generic"
	reportPrefix  = "waf-evaluation-report"
	payloadPrefix = "waf-evaluation-payloads"
)

var (
	configPath string
	verbose    bool
)

func main() {
	logger := log.New(os.Stdout, "GOTESTWAF : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	if err := run(logger); err != nil {
		logger.Println("main error:", err)
		os.Exit(1)
	}
}

func run(logger *log.Logger) error {
	parseFlags()
	if !verbose {
		logger.SetOutput(ioutil.Discard)
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		return errors.Wrap(err, "loading config")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Println("Test cases loading started")

	testCases, err := test.Load(cfg, logger)
	if err != nil {
		return errors.Wrap(err, "loading test case")
	}

	logger.Println("Test cases loading finished")

	db := test.NewDB(testCases)

	s := scanner.New(db, logger, cfg)

	logger.Println("Scanned URL:", cfg.URL)
	ok, httpStatus, err := s.PreCheck(cfg.URL)
	if err != nil {
		return errors.Wrap(err, "running pre-check")
	}
	if !ok {
		return errors.Errorf("WAF was not detected. "+
			"Please use the '--blockStatusCode' or '--blockRegex' flags. Use '--help' for additional info."+
			"\nBaseline attack status code: %v\n", httpStatus)
	}

	logger.Printf("WAF pre-check: OK. Blocking status code: %v\n", httpStatus)

	// If WS URL is not available - try to build it from WAF URL
	if cfg.WebSocketURL == "" {
		wsURL, wsErr := wsFromURL(cfg.URL)
		if wsErr != nil {
			logger.Printf("Can not parse WAF URL, reason: %s\n", wsErr)
			logger.Println("The provided WAF URL will be used for WebSocket testing")
		}
		cfg.WebSocketURL = wsURL
	}

	logger.Printf("WebSocket pre-check. URL to check: %s\n", cfg.WebSocketURL)

	available, blocked, err := s.WSPreCheck(cfg.WebSocketURL)
	if !available && err != nil {
		logger.Printf("WebSocket pre-check: connection is not available, "+
			"reason: %s\n", err)
	}
	if available && blocked {
		logger.Printf("WebSocket is available and payloads are "+
			"blocked by the WAF, reason: %s\n", err)
	}
	if available && !blocked {
		logger.Println("WebSocket is available and payloads are " +
			"not blocked by the WAF")
	}

	_, err = os.Stat(cfg.ReportPath)
	if os.IsNotExist(err) {
		if makeErr := os.Mkdir(cfg.ReportPath, 0700); makeErr != nil {
			return errors.Wrap(makeErr, "creating dir")
		}
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-shutdown
		logger.Printf("main: %v : scan canceled", sig)
		cancel()
	}()

	logger.Printf("Scanning %s\n", cfg.URL)
	err = s.Run(ctx, cfg.URL)
	if err != nil {
		return errors.Wrap(err, "run scanning")
	}

	reportTime := time.Now()
	reportSaveTime := reportTime.Format("2006-January-02-15-04-05")

	reportFile := filepath.Join(cfg.ReportPath, fmt.Sprintf("%s-%s-%s.pdf", reportPrefix, cfg.WAFName, reportSaveTime))

	rows, err := db.RenderTable(reportTime, cfg.WAFName)
	if err != nil {
		return errors.Wrap(err, "table rendering")
	}

	err = db.ExportToPDF(reportFile, reportTime, cfg.WAFName, cfg.URL, rows)
	if err != nil {
		return errors.Wrap(err, "PDF exporting")
	}

	payloadFiles := filepath.Join(cfg.ReportPath, fmt.Sprintf("%s-%s-%s.csv", payloadPrefix, cfg.WAFName, reportSaveTime))
	err = db.ExportPayloads(payloadFiles)
	if err != nil {
		errors.Wrap(err, "payloads exporting")
	}

	return nil
}

func parseFlags() {
	reportPath := filepath.Join(".", defaultReportPath)
	testCasesPath := filepath.Join(".", defaultTestCasesPath)

	flag.StringVar(&configPath, "configPath", defaultConfigPath, "Path to the config file")
	flag.BoolVar(&verbose, "verbose", true, "If true, enable verbose logging")

	flag.String("url", "http://localhost/", "URL to check")
	flag.String("wsURL", "", "WebSocket URL to check")
	flag.String("proxy", "", "Proxy URL to use")
	flag.Bool("tlsVerify", false, "If true, the received TLS certificate will be verified")
	flag.Int("maxIdleConns", 2, "The maximum number of keep-alive connections")
	flag.Int("maxRedirects", 50, "The maximum number of handling redirects")
	flag.Int("idleConnTimeout", 2, "The maximum amount of time a keep-alive connection will live")
	flag.Bool("followCookies", false, "If true, use cookies sent by the server. May work only with --maxIdleConns=1")
	flag.Int("blockStatusCode", 403, "HTTP status code that WAF uses while blocking requests")
	flag.Int("passStatusCode", 200, "HTTP response status code that WAF uses while passing requests")
	flag.String("blockRegex", "",
		"Regex to detect a blocking page with the same HTTP response status code as a not blocked request")
	flag.String("passRegex", "",
		"Regex to a detect normal (not blocked) web page with the same HTTP status code as a blocked request")
	flag.Bool("nonBlockedAsPassed", false,
		"If true, count requests that weren't blocked as passed. If false, requests that don't satisfy to PassStatuscode/PassRegExp as blocked")
	flag.Int("workers", 200, "The number of workers to scan")
	flag.Int("sendDelay", 400, "Delay in ms between requests")
	flag.Int("randomDelay", 400, "Random delay in ms in addition to the delay between requests")
	flag.String("testCase", "", "If set then only this test case will be run")
	flag.String("testSet", "", "If set then only this test set's cases will be run")
	flag.String("reportPath", reportPath, "A directory to store reports")
	flag.String("testCasesPath", testCasesPath, "Path to a folder with test cases")
	flag.String("wafName", wafName, "Name of the WAF product")
	flag.Parse()
}

// loadConfig loads the specified config file and merges it with the parameters passed via CLI.
func loadConfig(path string) (cfg *config.Config, err error) {
	err = viper.BindPFlags(flag.CommandLine)
	if err != nil {
		return nil, err
	}
	viper.AddConfigPath(".")
	viper.SetConfigFile(path)
	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}
	err = viper.Unmarshal(&cfg)
	return
}

func wsFromURL(wafURL string) (string, error) {
	urlParse, err := url.Parse(wafURL)
	if err != nil {
		return "", err
	}
	wsScheme := "ws"
	if urlParse.Scheme == "https" {
		wsScheme = "wss"
	}
	urlParse.Scheme = wsScheme
	return urlParse.String(), nil
}
