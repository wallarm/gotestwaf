package integration

import (
	"context"
	"log"
	"os"
	"path"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/data/config"
	"github.com/wallarm/gotestwaf/internal/data/test"
	"github.com/wallarm/gotestwaf/internal/scanner"
	"github.com/wallarm/gotestwaf/tests/integration/waf"

	test_config "github.com/wallarm/gotestwaf/tests/integration/config"
)

var (
	reportPrefix  = "waf-evaluation-report"
	payloadPrefix = "waf-evaluation-payloads"
)

func TestGoTestWAF(t *testing.T) {
	done := make(chan bool)
	errChan := make(chan error)

	testCases, allTestCases := test_config.GenerateTestCases()

	w := waf.New(errChan, allTestCases)

	go func() {
		err := w.Run()
		if err != nil {
			errChan <- err
		}
	}()

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

	tempDir := os.TempDir()

	cfg := &config.Config{
		Cookies:            nil,
		URL:                "http://" + test_config.Address,
		WebSocketURL:       "",
		HTTPHeaders:        nil,
		TLSVerify:          false,
		Proxy:              "",
		MaxIdleConns:       2,
		MaxRedirects:       50,
		IdleConnTimeout:    2,
		FollowCookies:      false,
		BlockStatusCode:    403,
		PassStatusCode:     203,
		BlockRegex:         "",
		PassRegex:          "",
		NonBlockedAsPassed: false,
		Workers:            1,
		RandomDelay:        400,
		SendDelay:          200,
		ReportPath:         path.Join(tempDir, "reports"),
		TestCase:           "",
		TestCasesPath:      "",
		TestSet:            "",
		WAFName:            "test-waf",
		IgnoreUnresolved:   false,
		BlockConnReset:     false,
		SkipWAFBlockCheck:  false,
		AddHeader:          "",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := test.NewDB(testCases)
	httpClient, err := scanner.NewHTTPClient(cfg)
	if err != nil {
		return errors.Wrap(err, "HTTP client")
	}

	grpcData, err := scanner.NewGRPCData(cfg)
	if err != nil {
		return errors.Wrap(err, "gRPC client")
	}

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

	logger.Printf("Scanning %s\n", cfg.URL)
	err = s.Run(ctx, cfg.URL, cfg.BlockConnReset)
	if err != nil {
		return errors.Wrap(err, "run scanning")
	}

	reportTime := time.Now()
	// reportSaveTime := reportTime.Format("2006-January-02-15-04-05")

	// reportFile := filepath.Join(cfg.ReportPath, fmt.Sprintf("%s-%s-%s.pdf", reportPrefix, cfg.WAFName, reportSaveTime))

	_ /*rows*/, err = db.RenderTable(reportTime, cfg.WAFName, cfg.IgnoreUnresolved)
	if err != nil {
		return errors.Wrap(err, "table rendering")
	}

	// err = db.ExportToPDF(reportFile, reportTime, cfg.WAFName, cfg.URL, rows, cfg.IgnoreUnresolved)
	// if err != nil {
	// 	return errors.Wrap(err, "PDF exporting")
	// }

	// payloadFiles := filepath.Join(cfg.ReportPath, fmt.Sprintf("%s-%s-%s.csv", payloadPrefix, cfg.WAFName, reportSaveTime))
	// err = db.ExportPayloads(payloadFiles)
	// if err != nil {
	// 	return errors.Wrap(err, "payloads exporting")
	// }

	return nil
}
