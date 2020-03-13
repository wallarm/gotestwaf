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
	blockStatuscode := flag.Int("block_statuscode", 403, "HTTP response status code that WAF use while blocking requests. 403 by default")
	reportFile := flag.String("report", "/tmp/report/waf-test-report"+current.Format("2006-January-02")+".pdf", "Report filename to export results.")

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
		conf.BlockStatusCode = *blockStatuscode
	}
	if conf.ReportFile == "" {
		conf.ReportFile = *reportFile
	}

	check, status := testcase.PreCheck(*url, conf)
	if !check {
		fmt.Printf("[FATAL] WAF was not detected. Please check the 'block_statuscode' value. Baseline attack returned: %v", status)
		return
	}

	os.Mkdir("/tmp/report", 0700)
	fmt.Printf("Checking %s\n", *url)

	results := testcase.Run(*url, conf)

	results.ExportPDF(conf.ReportFile)
}
