package integration

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/data/test"
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

	t.Cleanup(func() {
		err := w.Shutdown()
		if err != nil {
			t.Logf("WAF shutdown error: %v", err)
		}
	})

	go func() {
		err := runGoTestWAF(testCases)
		if err != nil {
			errChan <- err
		} else {
			done <- true
		}
	}()

	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("got an error during the test: %v", err)
		}
	case <-done:
		if allTestCases.CountTestCases() != 0 {
			t.Fatal("not all tests cases were processed")
		}
	}
}

func runGoTestWAF(testCases []test.Case) error {
	logger := log.New(os.Stdout, "GOTESTWAF : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	cfg := test_config.GetConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := test.NewDB(testCases)
	httpClient, err := scanner.NewHTTPClient(cfg)
	if err != nil {
		return errors.Wrap(err, "HTTP client")
	}

	cfg.URL = "http://" + test_config.GRPCAddress

	grpcData, err := scanner.NewGRPCData(cfg)
	if err != nil {
		return errors.Wrap(err, "gRPC client")
	}

	cfg.URL = "http://" + test_config.HTTPAddress

	logger.Printf("gRPC pre-check: IN PROGRESS")

	available, err := grpcData.CheckAvailability()
	if err != nil {
		logger.Printf("gRPC pre-check: connection is not available, "+
			"reason: %s\n", err)
	}
	if available {
		logger.Printf("gRPC pre-check: OK")
		grpcData.SetAvailability(available)
	}

	s := scanner.New(db, logger, cfg, httpClient, grpcData, true)

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
	err = s.Run(ctx, cfg.URL, cfg.BlockConnReset)
	if err != nil {
		return errors.Wrap(err, "run scanning")
	}

	reportTime := time.Now()

	_, err = db.RenderTable(reportTime, cfg.WAFName, cfg.IgnoreUnresolved)
	if err != nil {
		return errors.Wrap(err, "table rendering")
	}

	return nil
}
