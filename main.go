package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/ardanlabs/conf"
	"github.com/pkg/errors"
	"github.com/wallarm/gotestwaf/config"
	"github.com/wallarm/gotestwaf/scanner"
	"gopkg.in/yaml.v2"
)

const reportPrefix = "waf-evaluation-report"
const payloadPrefix = "waf-evaluation-payloads"

func main() {
	logger := log.New(os.Stdout, "GOTESTWAF : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	if err := run(logger); err != nil {
		log.Println("main: error:", err)
		os.Exit(1)
	}
}

func run(logger *log.Logger) error {
	var cfg config.Config

	if err := conf.Parse(os.Args[1:], "GOTESTWAF", &cfg); err != nil {
		switch err {
		case conf.ErrHelpWanted:
			usage, err := conf.Usage("GOTESTWAF", &cfg)
			if err != nil {
				return errors.Wrap(err, "generating config usage")
			}
			fmt.Println(usage)
			return nil
		case conf.ErrVersionWanted:
			version, err := conf.VersionString("GOTESTWAF", &cfg)
			if err != nil {
				return errors.Wrap(err, "generating config version")
			}
			fmt.Println(version)
			return nil
		}
		return errors.Wrap(err, "parsing config")
	}

	yamlFile, err := ioutil.ReadFile(cfg.YAMLConfigPath)
	if err != nil {
		return errors.Wrap(err, "failed to read config file")
	}

	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshall config file")
	}

	s := scanner.New(&cfg, logger)
	check, status, err := s.PreCheck(cfg.URL)
	if err != nil {
		return errors.Wrap(err, "running pre-check")
	}
	if !check {
		return errors.Errorf("WAF was not detected. "+
			"Please check the 'block_statuscode' or 'block_regexp' values."+
			"\nBaseline attack status code: %v\n", status)
	}

	logger.Printf("WAF pre-check: OK. Blocking status code: %v\n", status)

	if _, err := os.Stat(cfg.ReportDir); os.IsNotExist(err) {
		if err := os.Mkdir(cfg.ReportDir, 0700); err != nil {
			return errors.Wrap(err, "creating dir")
		}
	}

	logger.Printf("Checking %s\n", cfg.URL)
	report, err := s.Run(cfg.URL)
	if err != nil {
		return errors.Wrap(err, "running tests")
	}

	reportFile := cfg.ReportDir+"/"+reportPrefix+"-"+time.Now().Format("2006-January-02-11-06")+".pdf"
	err = report.ExportToPDFAndShowTable(reportFile)
	if err != nil {
		return errors.Wrap(err, "exporting report")
	}

	payloadFiles := cfg.ReportDir+"/"+payloadPrefix+"-"+time.Now().Format("2006-January-02-11-06")+".csv"
	err = report.ExportPayloads(payloadFiles)
	if err != nil {
		return errors.Wrap(err, "exporting payloads")
	}
	return nil
}
