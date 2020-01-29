package main

import (
	"flag"
	"fmt"

	"gotestwaf/config"
	"gotestwaf/report"
	"gotestwaf/testcase"
)

func main() {
	// TODO: understand why it doesn't work
	url := flag.String("url", "http://localhost", "URL with a WAF to check")
	configFile := flag.String("config", "config.yaml", "Config file to use")

	config := config.LoadConfig(*configFile)

	//url := os.Args[1]
	flag.Parse()

	fmt.Printf("Testing some WAF at the URL: %s\n", *url)

	results := testcase.Run(*url, config)
	report.ExportPDF(results)
}
