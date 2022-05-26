package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/db"
	"github.com/wallarm/gotestwaf/internal/openapi"
	"github.com/wallarm/gotestwaf/internal/report"
	"github.com/wallarm/gotestwaf/internal/scanner"
	"github.com/wallarm/gotestwaf/internal/version"
)

func main() {
	logger := log.New(os.Stdout, "GOTESTWAF : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-shutdown
		logger.Printf("main: %v : scan canceled", sig)
		cancel()
	}()

	if err := run(ctx, logger); err != nil {
		logger.Println("main error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *log.Logger) error {
	err := parseFlags()
	if err != nil {
		return err
	}

	if !verbose {
		logger.SetOutput(ioutil.Discard)
	}

	cfg, err := loadConfig()
	if err != nil {
		return errors.Wrap(err, "couldn't load config")
	}

	logger.Printf("GoTestWAF %s\n", version.Version)

	var openapiDoc *openapi3.T
	var router routers.Router
	var templates openapi.Templates

	if cfg.OpenAPIFile != "" {
		openapiDoc, router, err = openapi.LoadOpenAPISpec(ctx, cfg.OpenAPIFile)
		if err != nil {
			return errors.Wrap(err, "couldn't load OpenAPI spec")
		}
		openapiDoc.Servers = append(openapiDoc.Servers, &openapi3.Server{
			URL: cfg.URL,
		})

		templates, err = openapi.NewTemplates(openapiDoc, cfg.URL)
		if err != nil {
			return errors.Wrap(err, "couldn't create templates from OpenAPI file")
		}
	}

	logger.Println("Test cases loading started")
	testCases, err := db.LoadTestCases(cfg)
	if err != nil {
		return errors.Wrap(err, "couldn't load test case")
	}
	logger.Println("Test cases loading finished")

	db := db.NewDB(testCases)

	s, err := scanner.New(logger, cfg, db, templates, router, false)
	if err != nil {
		return errors.Wrap(err, "couldn't create scanner")
	}

	err = s.WAFBlockCheck()
	if err != nil {
		return err
	}

	s.WAFwsBlockCheck()
	s.CheckGRPCAvailability()

	err = s.Run(ctx)
	if err != nil {
		return errors.Wrap(err, "error occurred while scanning")
	}

	_, err = os.Stat(cfg.ReportPath)
	if os.IsNotExist(err) {
		if makeErr := os.Mkdir(cfg.ReportPath, 0700); makeErr != nil {
			return errors.Wrap(makeErr, "creating dir")
		}
	}

	reportTime := time.Now()
	reportName := reportTime.Format(cfg.ReportName)

	reportFile := filepath.Join(cfg.ReportPath, reportName)
	if cfg.RenderToHTML {
		reportFile += ".html"
	} else {
		reportFile += ".pdf"
	}

	stat := db.GetStatistics(cfg.IgnoreUnresolved, cfg.NonBlockedAsPassed)
	report.RenderConsoleTable(stat, reportTime, wafName, cfg.IgnoreUnresolved)
	err = report.ExportToPDF(stat, reportFile, reportTime, cfg.WAFName, cfg.URL, cfg.IgnoreUnresolved, cfg.RenderToHTML)
	if err != nil {
		return errors.Wrap(err, "PDF exporting")
	}
	fmt.Printf("\nreport is ready: %s\n", reportFile)

	payloadFiles := filepath.Join(cfg.ReportPath, reportName+".csv")
	err = db.ExportPayloads(payloadFiles)
	if err != nil {
		errors.Wrap(err, "payloads exporting")
	}

	return nil
}
