package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/wallarm/gotestwaf/internal/db"
	"github.com/wallarm/gotestwaf/internal/helpers"
	"github.com/wallarm/gotestwaf/internal/openapi"
	"github.com/wallarm/gotestwaf/internal/report"
	"github.com/wallarm/gotestwaf/internal/scanner"
	"github.com/wallarm/gotestwaf/internal/version"
)

func main() {
	logger := logrus.New()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-shutdown
		logger.WithField("signal", sig).Info("scan canceled")
		cancel()
	}()

	if err := run(ctx, logger); err != nil {
		logger.WithError(err).Error("caught error in main function")
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *logrus.Logger) error {
	args, err := parseFlags()
	if err != nil {
		return err
	}

	if quiet {
		logger.SetOutput(io.Discard)
	}
	logger.SetLevel(logLevel)

	if logFormat == jsonLogFormat {
		logger.SetFormatter(&logrus.JSONFormatter{})
	}

	cfg, err := loadConfig()
	if err != nil {
		return errors.Wrap(err, "couldn't load config")
	}

	logger.WithField("version", version.Version).Info("GoTestWAF started")

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

	logger.Info("Test cases loading started")

	testCases, err := db.LoadTestCases(cfg)
	if err != nil {
		return errors.Wrap(err, "loading test case")
	}

	logger.Info("Test cases loading finished")

	db, err := db.NewDB(testCases)
	if err != nil {
		return errors.Wrap(err, "couldn't create test cases DB")
	}

	logger.WithField("fp", db.Hash).Info("Test cases fingerprint")

	dnsCache, err := scanner.NewDNSCache(logger)
	if err != nil {
		return errors.Wrap(err, "couldn't create DNS cache")
	}

	if !cfg.SkipWAFIdentification {
		detector, err := scanner.NewDetector(cfg, dnsCache)
		if err != nil {
			return errors.Wrap(err, "couldn't create WAF detector")
		}

		logger.Info("Try to identify WAF solution")

		name, vendor, err := detector.DetectWAF(ctx)
		if err != nil {
			return errors.Wrap(err, "couldn't detect")
		}

		if name != "" && vendor != "" {
			logger.WithFields(logrus.Fields{
				"solution": name,
				"vendor":   vendor,
			}).Info("WAF was identified. Force enabling `--followCookies' and `--renewSession' options")

			cfg.FollowCookies = true
			cfg.RenewSession = true
			cfg.WAFName = fmt.Sprintf("%s (%s)", name, vendor)
		} else {
			logger.Info("WAF was not identified")
		}
	}

	s, err := scanner.New(logger, cfg, db, dnsCache, templates, router, cfg.AddDebugHeader)
	if err != nil {
		return errors.Wrap(err, "couldn't create scanner")
	}

	err = s.WAFBlockCheck(ctx)
	if err != nil {
		return err
	}

	s.WAFwsBlockCheck(ctx)
	s.CheckGRPCAvailability(ctx)

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

	stat := db.GetStatistics(cfg.IgnoreUnresolved, cfg.NonBlockedAsPassed)

	err = report.RenderConsoleReport(stat, reportTime, cfg.WAFName, cfg.URL, args, cfg.IgnoreUnresolved, logFormat)
	if err != nil {
		return err
	}

	if cfg.ReportFormat == report.NoneFormat {
		return nil
	}

	includePayloads := cfg.IncludePayloads
	if cfg.ReportFormat == report.HtmlFormat || cfg.ReportFormat == report.PdfFormat {
		askForPayloads := true

		// If the cfg.IncludePayloads is already explicitly set by the user OR
		// the user has explicitly chosen not to send email report, or has
		// provided the email to send the report to (which we interpret as
		// non-interactive mode), do not ask to include the payloads in the report.
		if isIncludePayloadsFlagUsed || cfg.NoEmailReport || cfg.Email != "" {
			askForPayloads = false
		}

		if askForPayloads {
			input := ""
			fmt.Print("Do you want to include payload details to the report? ([y/N]): ")
			fmt.Scanln(&input)

			if strings.TrimSpace(input) == "y" {
				includePayloads = true
			}
		}
	}

	reportFile, err = report.ExportFullReport(
		ctx, stat, reportFile,
		reportTime, cfg.WAFName, cfg.URL, cfg.OpenAPIFile, args,
		cfg.IgnoreUnresolved, includePayloads, cfg.ReportFormat,
	)
	if err != nil {
		return errors.Wrap(err, "couldn't export full report")
	}

	logger.WithField("filename", reportFile).Infof("Export full report")

	payloadFiles := filepath.Join(cfg.ReportPath, reportName+".csv")
	err = db.ExportPayloads(payloadFiles)
	if err != nil {
		errors.Wrap(err, "payloads exporting")
	}

	if !cfg.NoEmailReport {
		email := ""

		if cfg.Email != "" {
			email = cfg.Email
		} else {
			fmt.Print("Email to send the report (ENTER to skip): ")
			fmt.Scanln(&email)

			email = strings.TrimSpace(email)
			if email == "" {
				logger.Info("Skip report sending to email")

				return nil
			}

			email, err = helpers.ValidateEmail(email)
			if err != nil {
				return errors.Wrap(err, "couldn't validate email")
			}
		}

		err = report.SendReportByEmail(
			ctx, stat, email,
			reportTime, cfg.WAFName, cfg.URL, cfg.OpenAPIFile, args,
			cfg.IgnoreUnresolved, includePayloads,
		)
		if err != nil {
			return errors.Wrap(err, "couldn't send report by email")
		}

		logger.WithField("email", email).Info("The report has been sent to the specified email")
	}

	return nil
}
