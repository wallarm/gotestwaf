package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"gotestwaf/config"
	"gotestwaf/testcase"
)

func main() {

	current := time.Now()

	url := flag.String("url", "http://localhost", "URL with a WAF to check")
	configFile := flag.String("config", "config.yaml", "Config file to use. Attention, if you are using the config, all the are flags will be avoided.")
	testcasesFolder := flag.String("testcases", "./testcases/", "Folder with test cases")
	proxyUrl := flag.String("proxy", "", "Proxy to use")
	threads := flag.Int("threads", 2, "Number of concurrent HTTP requests")
	checkCertificates := flag.Bool("check_cert", false, "Check SSL/TLS certificates, turned off by default")
	blockStatusCode := flag.Int("block_statuscode", 403, "HTTP response status code that WAF use while blocking requests. 403 by default")
	blockRegExp := flag.String("block_regexp", "", "Regular Expression to detect blocking page with the same HTTP response status code as not blocked request")
	passStatusCode := flag.Int("pass_statuscode", 200, "HTTP response status code that WAF use while passing requests. 200 by default")
	passRegExp := flag.String("pass_regexp", "", "Regular Expression to detect normal (not blocked) web-page with the same HTTP response status code as blocked request")
	reportFile := flag.String("report", "/tmp/report/waf-test-report"+current.Format("2006-January-02")+".pdf", "Report filename to export results")
	nonBlockedAsPassed := flag.Bool("nonblocked_as_passed", true, "Count all the requests not blocked as passed (old behaviour). Otherwise, count all of them that doens't satisfy PassStatuscode/PassRegExp as blocked (by default)")

	flag.Parse()

	conf := config.LoadConfig(*configFile)

	if conf.TestcasesFolder == "" {
		conf.TestcasesFolder = *testcasesFolder
	}
	if conf.Proxy == "" {
		conf.Proxy = *proxyUrl
	}
	if conf.MaxIddleConnections == 0 {
		conf.MaxIddleConnections = *threads
	}
	if !conf.CertificateCheck {
		conf.CertificateCheck = *checkCertificates
	}
	if conf.BlockStatusCode == 0 {
		conf.BlockStatusCode = *blockStatusCode
	}
	if conf.BlockRegExp == "" {
		conf.BlockRegExp = *blockRegExp
	}
	if conf.PassStatusCode == 0 {
		conf.PassStatusCode = *passStatusCode
	}
	if conf.PassRegExp == "" {
		conf.PassRegExp = *passRegExp
	}
	if conf.ReportFile == "" {
		conf.ReportFile = *reportFile
	}
	if !conf.NonBlockedAsPassed {
		conf.NonBlockedAsPassed = *nonBlockedAsPassed
	}

	check, status := testcase.PreCheck(*url, conf)
	if !check {
		fmt.Printf("[FATAL] WAF was not detected. Please check the 'block_statuscode' or 'block_regexp' values.\nBaseline attack status code: %v\n", status)
		return
	} else {
		fmt.Printf("WAF pre-check: OK. Blocking status code: %v\n", status)
	}

	os.Mkdir("/tmp/report", 0700)
	fmt.Printf("Checking %s\n", *url)

	results := testcase.Run(*url, conf)

	results.ExportPDF(conf.ReportFile)
}
