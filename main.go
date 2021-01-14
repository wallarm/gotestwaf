package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
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
	checkCertificates := flag.Bool("check-cert", false, "Check SSL/TLS certificates, turned off by default")
	blockStatusCode := flag.Int("block-statuscode", 403, "HTTP response status code that WAF use while blocking requests. 403 by default")
	blockRegExp := flag.String("block-regexp", "", "Regular Expression to detect blocking page with the same HTTP response status code as not blocked request")
	passStatusCode := flag.Int("pass-statuscode", 200, "HTTP response status code that WAF use while passing requests. 200 by default")
	passRegExp := flag.String("pass-regexp", "", "Regular Expression to detect normal (not blocked) web-page with the same HTTP response status code as blocked request")
	reportFile := flag.String("report", "/tmp/report/waf-evaluation-report-"+current.Format("2006-January-02")+".pdf", "PDF report filename used to export results")
	nonBlockedAsPassed := flag.Bool("nonblocked_as_passed", false, "Count all the requests that were not blocked as passed (old behavior). Otherwise, count all of them that doesn't satisfy PassStatuscode/PassRegExp as blocked (by default)")
	followCookies := flag.Bool("follow-cookies", false, "Allow GoTestWAF to use cookies server sent. May work only for --threads=1. Default: false")
	maxRedirects := flag.Int("max-redirects", 50, "Maximum amount of redirects per request that GoTestWAF will follow until the hard stop. Default: 50")
	sendingDelay := flag.Int("sending-delay", 500, "Delay between sending requests inside threads, milliseconds. Default: 500ms")
	randomDelay := flag.Int("random-delay", 500, "Random delay, in addition to --sending_delay between requests inside threads, milliseconds. Default: up to +500ms")
	headers := flag.String("headers", "", "The list of HTTP headers to add to each request, separated by ',' (comma). Example: -headers=X-a:aaa,X-b:bbb. Clear the config.yaml headers section prior to using this option.")
	payloadsExportFile := flag.String("payloads-export", "/tmp/report/waf-payloads-export-"+current.Format("2006-January-02")+".csv", "Export payloads to the text file for reusing later, i.e. for Burp Suite pasting.")

	flag.Parse()

	conf := config.LoadConfig(*configFile)

	if len(conf.Headers) == 0 {
		conf.Headers = make(map[string]string)
	}

	for _, h := range strings.Split(*headers, ",") {
		header := strings.Split(h, ":")
		if len(header) == 2 {
			conf.Headers[header[0]] = header[1]
		}
	}

	/*setting up limits on some values*/
	if *randomDelay <= 0 {
		*randomDelay = 1 //otherwise it will cause panic at rand.Intn()
	}

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
	if conf.PayloadsExportFile == "" {
		conf.PayloadsExportFile = *payloadsExportFile
	}
	if !conf.NonBlockedAsPassed {
		conf.NonBlockedAsPassed = *nonBlockedAsPassed
	}
	if !conf.FollowCookies {
		conf.FollowCookies = *followCookies
	}
	if conf.MaxRedirects == 0 {
		conf.MaxRedirects = *maxRedirects
	}
	if conf.SendingDelay == 0 {
		conf.SendingDelay = *sendingDelay
	}
	if conf.RandomDelay <= 0 {
		conf.RandomDelay = *randomDelay
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
	results.ExportPayloads(conf.PayloadsExportFile)
}
