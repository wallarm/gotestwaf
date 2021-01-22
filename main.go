package main

import (
	"log"
	"os"
	"time"

	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/wallarm/gotestwaf/config"
	"github.com/wallarm/gotestwaf/scanner"
)

const (
	reportPrefix  = "waf-evaluation-report"
	payloadPrefix = "waf-evaluation-payloads"
)

var (
	configPath string
)

func main() {
	logger := log.New(os.Stdout, "GOTESTWAF : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	if err := run(logger); err != nil {
		logger.Println("main: error:", err)
		os.Exit(1)
	}
}

func run(logger *log.Logger) error {
	parseFlags()

	cfg, err := loadConfig(configPath)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Println("Scanned URL:", cfg.URL)

	s := scanner.New(cfg, logger)

	ok, httpStatus, err := s.PreCheck(cfg.URL)
	if err != nil {
		return errors.Wrap(err, "running pre-check")
	}
	if !ok {
		return errors.Errorf("WAF was not detected. "+
			"Please check the 'block_statuscode' or 'block_regexp' values."+
			"\nBaseline attack status code: %v\n", httpStatus)
	}

	logger.Printf("WAF pre-check: OK. Blocking status code: %v\n", httpStatus)

	_, err = os.Stat(cfg.ReportDir)
	if os.IsNotExist(err) {
		if makeErr := os.Mkdir(cfg.ReportDir, 0700); makeErr != nil {
			return errors.Wrap(makeErr, "creating dir")
		}
	}

	logger.Printf("Checking %s\n", cfg.URL)
	report, order, err := s.Run(cfg.URL)
	if err != nil {
		return errors.Wrap(err, "running tests")
	}

	reportFile := cfg.ReportDir + "/" + reportPrefix + "-" + time.Now().Format("2006-January-02-11-06") + ".pdf"
	err = report.ExportToPDFAndShowTable(reportFile, order)
	if err != nil {
		return errors.Wrap(err, "exporting report")
	}

	payloadFiles := cfg.ReportDir + "/" + payloadPrefix + "-" + time.Now().Format("2006-January-02-11-06") + ".csv"
	err = report.ExportPayloads(payloadFiles)
	if err != nil {
		return errors.Wrap(err, "exporting payloads")
	}
	return nil
}

func parseFlags() {
	flag.StringVar(&configPath, "configPath", "config.yaml", "Path to a config file")
	flag.String("url", "http://localhost/", "URL to check")
	flag.String("proxy", "", "Proxy URL to use")
	flag.Bool("tlsverify", false, "If true, the received TLS certificate will be verified")
	flag.Int("maxIdleConns", 2, "The maximum amount of time a keep-alive connection will live")
	flag.Int("maxRedirects", 50, "The maximum number of handling redirects")
	flag.Int("idleConnTimeout", 2, "The maximum number of keep-alive connections")
	flag.Bool("followCookies", false, "If true, use cookies sent by the server. May work only with --maxIdleConns=1")
	flag.Int("blockStatusCode", 403, "HTTP status code that WAF uses while blocking requests")
	flag.Int("passStatusCode", 200, "HTTP response status code that WAF uses while passing requests")
	flag.String("blockRegex", "", "Regex to detect a blocking page with the same HTTP response status code as a not blocked request")
	flag.String("passRegex", "", "Regex to a detect normal (not blocked) web page with the same HTTP status code as a blocked request")
	flag.Bool("nonBlockedAsPassed", false, "If true, count requests that weren't blocked as passed. If false, requests that don't satisfy to PassStatuscode/PassRegExp as blocked")
	flag.Int("sendDelay", 500, "Delay in ms between requests")
	flag.Int("randomDelay", 500, "Random delay in ms in addition to --sendDelay")
	flag.String("testCasesPath", "./testcases/", "Path to a folder with test cases")
	flag.String("reportDir", "/tmp/gotestwaf/", "A directory to store reports")

	flag.Parse()
}

func loadConfig(configPath string) (cfg *config.Config, err error) {
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
