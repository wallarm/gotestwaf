package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/wallarm/gotestwaf/internal/db"
	"github.com/wallarm/gotestwaf/internal/report"
	"github.com/wallarm/gotestwaf/internal/scanner"
	"github.com/wallarm/gotestwaf/tests/integration/waf"

	test_config "github.com/wallarm/gotestwaf/tests/integration/config"
)

func TestGoTestWAF(t *testing.T) {
	done := make(chan bool)
	errChan := make(chan error)

	err := test_config.PickUpTestPorts()
	if err != nil {
		t.Fatalf("couldn't pickup two tcp ports for testing: %s", err)
	}

	testCases, allTestCases := test_config.GenerateTestCases()

	w := waf.New(errChan, allTestCases)

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
		err := runGoTestWAF(ctx, testCases)
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
		}
	case <-done:
		if allTestCases.CountTestCases() != 0 {
			remaining := allTestCases.GetRemainingValues()
			t.Fatalf("not all tests cases were processed: %v", remaining)
		}
	}
}

func runGoTestWAF(ctx context.Context, testCases []*db.Case) error {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	cfg := test_config.GetConfig()

	db := db.NewDB(testCases)

	s, err := scanner.New(logger, cfg, db, nil, nil, false)
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

	stat := db.GetStatistics(cfg.IgnoreUnresolved, cfg.NonBlockedAsPassed)

	err = report.RenderConsoleReport(stat, reportTime, cfg.WAFName, cfg.URL, cfg.IgnoreUnresolved, "text")
	if err != nil {
		return err
	}

	return nil
}
