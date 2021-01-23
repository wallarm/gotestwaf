package cmd

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/wallarm/gotestwaf/internal/data/config"
	"github.com/wallarm/gotestwaf/internal/data/test"
	"github.com/wallarm/gotestwaf/internal/scanner"
)

const (
	reportPrefix  = "waf-evaluation-report"
	payloadPrefix = "waf-evaluation-payloads"
)

var (
	configPath string
	verbose    bool
)

func Run() int {
	logger := log.New(os.Stdout, "GOTESTWAF : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	parseFlags()
	if !verbose {
		logger.SetOutput(ioutil.Discard)
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		logger.Println("loading config:", err)
		return 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Println("Test cases loading started")
	testCases, err := test.Load(cfg.TestCasesPath, logger)
	if err != nil {
		logger.Println("loading test cases:", err)
		return 1
	}
	logger.Println("Test cases loading finished")

	db := test.NewDB(testCases)

	s := scanner.New(db, logger, cfg)

	logger.Println("Scanned URL:", cfg.URL)
	ok, httpStatus, err := s.PreCheck(cfg.URL)
	if err != nil {
		logger.Println("running pre-check:", err)
		return 1
	}
	if !ok {
		logger.Printf("WAF was not detected. "+
			"Please check the 'block_statuscode' or 'block_regexp' values."+
			"\nBaseline attack status code: %v\n", httpStatus)
		return 1
	}

	logger.Printf("WAF pre-check: OK. Blocking status code: %v\n", httpStatus)

	_, err = os.Stat(cfg.ReportDir)
	if os.IsNotExist(err) {
		if makeErr := os.Mkdir(cfg.ReportDir, 0700); makeErr != nil {
			logger.Println("creating dir:", makeErr)
			return 1
		}
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	scannerErrors := make(chan error, 1)

	go func() {
		logger.Printf("Scanning %s\n", cfg.URL)
		scannerErrors <- s.Run(ctx, cfg.URL)
	}()

	select {
	case err = <-scannerErrors:
		if err != nil {
			logger.Println("scanner error:", err)
			return 1
		}

	case sig := <-shutdown:
		logger.Printf("main: %v : scan canceled", sig)
		cancel()
		return 0
	}

	reportFile := cfg.ReportDir + "/" + reportPrefix + "-" + time.Now().Format("2006-January-02-11-06") + ".pdf"
	err = db.ExportToPDFAndShowTable(reportFile)
	if err != nil {
		logger.Println("exporting report:", err)
		return 1
	}

	payloadFiles := cfg.ReportDir + "/" + payloadPrefix + "-" + time.Now().Format("2006-January-02-11-06") + ".csv"
	err = db.ExportPayloads(payloadFiles)
	if err != nil {
		logger.Println("exporting payloads:", err)
		return 1
	}
	return 0
}

func parseFlags() {
	flag.StringVar(&configPath, "configPath", "config.yaml", "Path to a config file")
	flag.BoolVar(&verbose, "verbose", true, "Enable verbose logging")

	flag.String("url", "http://localhost/", "URL to check")
	flag.String("proxy", "", "Proxy URL to use")
	flag.Bool("tlsverify", false, "If true, the received TLS certificate will be verified")
	flag.Int("maxIdleConns", 2, "The maximum number of keep-alive connections")
	flag.Int("maxRedirects", 50, "The maximum number of handling redirects")
	flag.Int("idleConnTimeout", 2, "The maximum amount of time a keep-alive connection will live")
	flag.Bool("followCookies", false, "If true, use cookies sent by the server. May work only with --maxIdleConns=1")
	flag.Int("blockStatusCode", 403, "HTTP status code that WAF uses while blocking requests")
	flag.Int("passStatusCode", 200, "HTTP response status code that WAF uses while passing requests")
	flag.String("blockRegex", "", "Regex to detect a blocking page with the same HTTP response status code as a not blocked request")
	flag.String("passRegex", "", "Regex to a detect normal (not blocked) web page with the same HTTP status code as a blocked request")
	flag.Bool("nonBlockedAsPassed", false, "If true, count requests that weren't blocked as passed. If false, requests that don't satisfy to PassStatuscode/PassRegExp as blocked")
	flag.Int("workers", 200, "The number of workers to scan")
	flag.Int("sendDelay", 400, "Delay in ms between requests")
	flag.Int("randomDelay", 400, "Random delay in ms in addition to --sendDelay")
	flag.String("testCasesPath", "./testcases/", "Path to a folder with test cases")
	flag.String("reportDir", "/tmp/gotestwaf/", "A directory to store reports")

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
