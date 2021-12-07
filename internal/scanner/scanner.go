package scanner

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"regexp"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/schollz/progressbar/v3"

	"github.com/wallarm/gotestwaf/internal/config"
	"github.com/wallarm/gotestwaf/internal/db"
	"github.com/wallarm/gotestwaf/internal/payload/encoder"
)

const (
	preCheckVector        = "<script>alert('union select password from users')</script>"
	wsPreCheckReadTimeout = time.Second * 1
)

type testWork struct {
	setName         string
	caseName        string
	payload         string
	encoder         string
	placeholder     string
	testType        string
	isTruePositive  bool
	testHeaderValue string
}

type Scanner struct {
	logger     *log.Logger
	cfg        *config.Config
	db         *db.DB
	httpClient *HTTPClient
	grpcConn   *GRPCConn
	wsClient   *websocket.Dialer
	isTestEnv  bool
}

func New(db *db.DB, logger *log.Logger, cfg *config.Config, httpClient *HTTPClient, grpcConn *GRPCConn, isTestEnv bool) *Scanner {
	return &Scanner{
		db:         db,
		logger:     logger,
		cfg:        cfg,
		httpClient: httpClient,
		grpcConn:   grpcConn,
		wsClient:   websocket.DefaultDialer,
		isTestEnv:  isTestEnv,
	}
}

func (s *Scanner) CheckBlocking(body []byte, statusCode int) (bool, error) {
	if s.cfg.BlockRegex != "" {
		m, _ := regexp.MatchString(s.cfg.BlockRegex, string(body))
		return m, nil
	}
	return statusCode == s.cfg.BlockStatusCode, nil
}

func (s *Scanner) CheckPass(body []byte, statusCode int) (bool, error) {
	if s.cfg.PassRegex != "" {
		m, _ := regexp.MatchString(s.cfg.PassRegex, string(body))
		return m, nil
	}
	return statusCode == s.cfg.PassStatusCode, nil
}

func (s *Scanner) BenignPreCheck(url string) (blocked bool, statusCode int, err error) {
	body, code, err := s.httpClient.Send(context.Background(), url, "URLParam", "URL", "", "")
	if err != nil {
		return false, 0, err
	}
	blocked, err = s.CheckBlocking(body, code)
	if err != nil {
		return false, 0, err
	}
	return blocked, code, nil
}

func (s *Scanner) PreCheck(url string) (blocked bool, statusCode int, err error) {
	body, code, err := s.httpClient.Send(context.Background(), url, "URLParam", "URL", preCheckVector, "")
	if err != nil {
		return false, 0, err
	}
	blocked, err = s.CheckBlocking(body, code)
	if err != nil {
		return false, 0, err
	}
	return blocked, code, nil
}

func (s *Scanner) WSPreCheck(url string) (available, blocked bool, err error) {
	wsClient, _, err := s.wsClient.Dial(url, nil)
	if err != nil {
		return false, false, err
	}
	defer wsClient.Close()

	wsPreCheckVectors := [...]string{
		fmt.Sprintf("{\"message\": \"%[1]s\", \"%[1]s\": \"%[1]s\"}", preCheckVector),
		preCheckVector,
	}

	block := make(chan error)
	receivedCtr := 0

	go func() {
		defer close(block)
		for {
			wsClient.SetReadDeadline(time.Now().Add(wsPreCheckReadTimeout))
			_, _, err := wsClient.ReadMessage()
			if err != nil {
				return
			}
			receivedCtr++
		}
	}()

	for i, payload := range wsPreCheckVectors {
		err = wsClient.WriteMessage(websocket.TextMessage, []byte(payload))
		if err != nil && i == 0 {
			return true, false, err
		} else if err != nil {
			return true, true, nil
		}
	}

	if _, open := <-block; !open && receivedCtr != len(wsPreCheckVectors) {
		return true, true, nil
	}

	return true, false, nil
}

func (s *Scanner) Run(ctx context.Context) error {
	gn := s.cfg.Workers
	var wg sync.WaitGroup
	wg.Add(gn)

	defer s.grpcConn.Close()

	rand.Seed(time.Now().UnixNano())

	s.logger.Println("Scanning started")
	defer s.logger.Println("Scanning finished")

	start := time.Now()
	defer s.logger.Println("Scanning Time: ", time.Since(start))

	testChan := s.produceTests(ctx, gn)

	bar := progressbar.NewOptions64(
		int64(s.db.GetNumberOfAllTestCases()),
		progressbar.OptionShowCount(),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionFullWidth(),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetDescription("Sending requests..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	for e := 0; e < gn; e++ {
		go func(ctx context.Context) {
			defer wg.Done()
			for {
				select {
				case w, ok := <-testChan:
					if !ok {
						return
					}
					time.Sleep(time.Duration(s.cfg.SendDelay+rand.Intn(s.cfg.RandomDelay)) * time.Millisecond)

					if err := s.scanURL(ctx, s.cfg.URL, s.cfg.BlockConnReset, w); err != nil {
						s.logger.Println(err)
					}
					bar.Add(1)
				case <-ctx.Done():
					return
				}
			}
		}(ctx)
	}

	wg.Wait()
	if errors.Is(ctx.Err(), context.Canceled) {
		return ctx.Err()
	}

	return nil
}

func (s *Scanner) produceTests(ctx context.Context, n int) <-chan *testWork {
	testChan := make(chan *testWork, n)
	testCases := s.db.GetTestCases()

	go func() {
		defer close(testChan)

		var testHeaderValue string

		for _, t := range testCases {
			for _, payload := range t.Payloads {
				for _, e := range t.Encoders {
					for _, placeholder := range t.Placeholders {
						if s.isTestEnv {
							testHeaderValue = fmt.Sprintf(
								"set=%s,name=%s,placeholder=%s,encoder=%s",
								t.Set, t.Name, placeholder, e,
							)
						} else {
							testHeaderValue = ""
						}
						wrk := &testWork{t.Set,
							t.Name,
							payload,
							e,
							placeholder,
							t.Type,
							t.IsTruePositive,
							testHeaderValue,
						}
						select {
						case testChan <- wrk:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}
	}()
	return testChan
}

func (s *Scanner) scanURL(ctx context.Context, url string, blockConn bool, w *testWork) error {
	var (
		body       []byte
		statusCode int
		err        error
	)

	switch w.encoder {
	case encoder.DefaultGRPCEncoder.GetName():
		if !s.grpcConn.IsAvailable() {
			return nil
		}

		body, statusCode, err = s.grpcConn.Send(ctx, w.encoder, w.payload)

	default:
		body, statusCode, err = s.httpClient.Send(ctx, url, w.placeholder, w.encoder, w.payload, w.testHeaderValue)
	}

	info := &db.Info{
		Set:                w.setName,
		Case:               w.caseName,
		Payload:            w.payload,
		Encoder:            w.encoder,
		Placeholder:        w.placeholder,
		ResponseStatusCode: statusCode,
		Type:               w.testType,
	}

	var blockedByReset bool
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, syscall.ECONNRESET) {
			if blockConn {
				blockedByReset = true
			} else {
				s.db.UpdateNaTests(info, s.cfg.IgnoreUnresolved, s.cfg.NonBlockedAsPassed, w.isTruePositive)
				return nil
			}
		} else {
			info.Reason = err.Error()
			s.db.UpdateFailedTests(info)
			s.logger.Printf("http sending: %s\n", err.Error())
			return nil
		}
	}

	var blocked, passed bool
	if blockedByReset {
		blocked = true
	} else {
		blocked, err = s.CheckBlocking(body, statusCode)
		if err != nil {
			return errors.Wrap(err, "failed to check blocking:")
		}

		passed, err = s.CheckPass(body, statusCode)
		if err != nil {
			return errors.Wrap(err, "failed to check passed or not:")
		}
	}

	if (blocked && passed) || (!blocked && !passed) {
		s.db.UpdateNaTests(info, s.cfg.IgnoreUnresolved, s.cfg.NonBlockedAsPassed, w.isTruePositive)
	} else {
		if blocked {
			s.db.UpdateBlockedTests(info)
		} else {
			s.db.UpdatePassedTests(info)
		}
	}
	return nil
}
