package integration

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/db"
	"github.com/wallarm/gotestwaf/internal/report"
	"github.com/wallarm/gotestwaf/internal/scanner"
	"github.com/wallarm/gotestwaf/tests/integration/waf"

	test_config "github.com/wallarm/gotestwaf/tests/integration/config"
)

func TestGoTestWAF(t *testing.T) {
	done := make(chan bool)
	errChan := make(chan error)

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

func runGoTestWAF(ctx context.Context, testCases []db.Case) error {
	logger := log.New(os.Stdout, "GOTESTWAF : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	cfg := test_config.GetConfig()

	db := db.NewDB(testCases)
	httpClient, err := scanner.NewHTTPClient(cfg)
	if err != nil {
		return errors.Wrap(err, "HTTP client")
	}

	cfg.URL = "http://" + test_config.GRPCAddress

	grpcConn, err := scanner.NewGRPCConn(cfg)
	if err != nil {
		return errors.Wrap(err, "gRPC client")
	}

	cfg.URL = "http://" + test_config.HTTPAddress

	logger.Printf("gRPC pre-check: IN PROGRESS")

	available, err := grpcConn.CheckAvailability()
	if err != nil {
		logger.Printf("gRPC pre-check: connection is not available, "+
			"reason: %s\n", err)
	}
	if available {
		logger.Printf("gRPC pre-check: GRPC IS AVAILABLE")
	} else {
		logger.Printf("gRPC pre-check: GRPC IS NOT AVAILABLE")
	}

	s := scanner.New(db, logger, cfg, httpClient, grpcConn, true)

	logger.Println("Scanned URL:", cfg.URL)

	_, err = os.Stat(cfg.ReportPath)
	if os.IsNotExist(err) {
		if makeErr := os.Mkdir(cfg.ReportPath, 0700); makeErr != nil {
			return errors.Wrap(makeErr, "creating dir")
		}
	}

	logger.Printf("WebSocket pre-check. URL to check: %s\n", cfg.WebSocketURL)

	available, blocked, err := s.WSPreCheck(cfg.WebSocketURL)
	if !available && err != nil {
		logger.Printf("WebSocket pre-check: connection is not available, "+
			"reason: %s\n", err)
	}
	if available && blocked {
		logger.Printf("WebSocket is available and payloads are "+
			"blocked by the WAF, reason: %s\n", err)
	}
	if available && !blocked {
		logger.Println("WebSocket is available and payloads are " +
			"not blocked by the WAF")
	}

	logger.Printf("Scanning %s\n", cfg.URL)
	err = s.Run(ctx)
	if err != nil {
		return errors.Wrap(err, "run scanning")
	}

	reportTime := time.Now()

	stat := db.GetStatistics(cfg.IgnoreUnresolved, cfg.NonBlockedAsPassed)
	report.RenderConsoleTable(stat, reportTime, "Test", cfg.IgnoreUnresolved)
	if err != nil {
		return errors.Wrap(err, "table rendering")
	}

	return nil
}
