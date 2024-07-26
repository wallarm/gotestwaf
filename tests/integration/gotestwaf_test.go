package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/wallarm/gotestwaf/internal/config"
	"github.com/wallarm/gotestwaf/internal/db"
	"github.com/wallarm/gotestwaf/internal/report"
	"github.com/wallarm/gotestwaf/internal/scanner"
	test_config "github.com/wallarm/gotestwaf/tests/integration/config"
	"github.com/wallarm/gotestwaf/tests/integration/waf"
)

func TestGoTestWAF(t *testing.T) {
	t.Run("GoHTTP client", func(t *testing.T) {
		httpPort, grpcPort, err := test_config.PickUpTestPorts()
		if err != nil {
			t.Fatalf("could not pick up test ports: %s", err)
		}

		runGoTestWAFTest(t, test_config.GetConfigWithGoHTTPClient(httpPort, grpcPort), httpPort, grpcPort)
	})

	t.Run("Chrome client", func(t *testing.T) {
		httpPort, grpcPort, err := test_config.PickUpTestPorts()
		if err != nil {
			t.Fatalf("could not pick up test ports: %s", err)
		}

		runGoTestWAFTest(t, test_config.GetConfigWithChromeClient(httpPort, grpcPort), httpPort, grpcPort)
	})
}

func runGoTestWAFTest(t *testing.T, cfg *config.Config, httpPort int, grpcPort int) {
	done := make(chan bool)
	errChan := make(chan error)

	testCases, allTestCases := test_config.GenerateTestCases()

	w := waf.New(errChan, allTestCases, httpPort, grpcPort)

	w.Run()

	ctx, cancel := context.WithCancel(context.Background())

	t.Cleanup(func() {
		cancel()
		err := w.Shutdown()
		if err != nil {
			t.Logf("WAF shutdown error: %v", err)
		}
	})

	go func() {
		err := runGoTestWAF(ctx, cfg, testCases)
		if err != nil {
			errChan <- err
		} else {
			done <- true
		}
	}()

	select {
	case err := <-errChan:
		if err != nil {
			cancel()
			t.Fatalf("got an error during the test: %v", err)
			return
		}
	case <-done:
		if allTestCases.CountTestCases() != 0 {
			remaining := allTestCases.GetRemainingValues()
			t.Fatalf("not all tests cases were processed: %v", remaining)
		}
	}
}

func runGoTestWAF(ctx context.Context, cfg *config.Config, testCases []*db.Case) error {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	db, err := db.NewDB(testCases)
	if err != nil {
		return errors.Wrap(err, "couldn't create test cases DB")
	}

	s, err := scanner.New(logger, cfg, db, nil, nil, cfg.AddDebugHeader)
	if err != nil {
		return errors.Wrap(err, "couldn't create scanner")
	}

	//s.CheckGRPCAvailability(ctx)
	//s.CheckGraphQLAvailability(ctx)

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

	stat := db.GetStatistics(cfg.IgnoreUnresolved, cfg.NonBlockedAsPassed)

	err = report.RenderConsoleReport(stat, reportTime, cfg.WAFName, cfg.URL, []string{""}, cfg.IgnoreUnresolved, "text")
	if err != nil {
		return err
	}

	return nil
}
