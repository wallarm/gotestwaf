package main

import (
	"flag"
	"fmt"

	"gotestwaf/config"
	"gotestwaf/testcase"
)

func main() {
	url := flag.String("url", "http://localhost", "URL with a WAF to check")
	configFile := flag.String("config", "config.yaml", "Config file to use")

	conf := config.LoadConfig(*configFile)
	flag.Parse()
	fmt.Printf("Checking %s", *url)

	results := testcase.Run(*url, conf)
	results.ExportPDF()
}
