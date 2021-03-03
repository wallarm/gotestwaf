package scanner

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/data/config"
	"github.com/wallarm/gotestwaf/internal/data/test"
	"github.com/wallarm/gotestwaf/internal/payload/encoder"
)

const preCheckVector = "<script>alert('union select password from users')</script>"

type testWork struct {
	setName        string
	caseName       string
	payload        string
	encoder        string
	placeholder    string
	isTruePositive bool
}

type Scanner struct {
	logger     *log.Logger
	cfg        *config.Config
	db         *test.DB
	httpClient *HTTPClient
	wsClient   *websocket.Dialer
}

func New(db *test.DB, logger *log.Logger, cfg *config.Config) *Scanner {
	encoder.InitEncoders()
	return &Scanner{
		db:         db,
		logger:     logger,
		cfg:        cfg,
		httpClient: NewHTTPClient(cfg),
		wsClient:   websocket.DefaultDialer,
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

func (s *Scanner) PreCheck(url string) (blocked bool, statusCode int, err error) {
	body, code, err := s.httpClient.Send(context.Background(), url, "URLParam", "URL", preCheckVector)
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

	wsPreCheckVectors := [...]string{
		fmt.Sprintf("{\"message\": \"%[1]s\", \"%[1]s\": \"%[1]s\"}", preCheckVector),
		preCheckVector,
	}

	wsError := make(chan error, 1)

	go func() {
		defer close(wsError)
		for {
			_, _, err := wsClient.ReadMessage()
			if err != nil {
				wsError <- err
			}
		}
	}()

	for _, payload := range wsPreCheckVectors {
		select {
		case err := <-wsError:
			return true, true, err
		default:
			err := wsClient.WriteMessage(websocket.TextMessage, []byte(payload))
			if err != nil {
				return true, true, err
			}
		}
	}

	return true, false, nil
}

func (s *Scanner) Run(ctx context.Context, url string) error {
	gn := s.cfg.Workers
	var wg sync.WaitGroup
	wg.Add(gn)

	rand.Seed(time.Now().UnixNano())

	s.logger.Println("Scanning started")
	defer s.logger.Println("Scanning finished")

	start := time.Now()
	defer func() {
		s.logger.Println("Scanning Time: ", time.Since(start))
	}()

	testChan := s.produceTests(ctx, gn)

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

					if err := s.scanURL(ctx, url, w); err != nil {
						s.logger.Println(err)
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}(ctx)
	}

	wg.Wait()
	if err := errors.Cause(ctx.Err()); err == context.Canceled {
		return ctx.Err()
	}
	return nil
}

func (s *Scanner) produceTests(ctx context.Context, n int) <-chan *testWork {
	testChan := make(chan *testWork, n)
	testCases := s.db.GetTestCases()

	go func() {
		defer close(testChan)
		for _, t := range testCases {
			for _, payload := range t.Payloads {
				for _, e := range t.Encoders {
					for _, placeholder := range t.Placeholders {
						wrk := &testWork{t.Set,
							t.Name,
							payload,
							e,
							placeholder,
							t.IsTruePositive,
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

func (s *Scanner) scanURL(ctx context.Context, url string, w *testWork) error {
	body, statusCode, err := s.httpClient.Send(ctx, url, w.placeholder, w.encoder, w.payload)
	if err != nil {
		return errors.Wrap(err, "http sending")
	}

	blocked, err := s.CheckBlocking(body, statusCode)
	if err != nil {
		return errors.Wrap(err, "failed to check blocking:")
	}

	passed, err := s.CheckPass(body, statusCode)
	if err != nil {
		return errors.Wrap(err, "failed to check passed or not:")
	}

	info := &test.Info{
		Set:                w.setName,
		Case:               w.caseName,
		Payload:            w.payload,
		Encoder:            w.encoder,
		Placeholder:        w.placeholder,
		ResponseStatusCode: statusCode,
	}
	if (blocked && passed) || (!blocked && !passed) {
		s.db.UpdateNaTests(info, s.cfg.NonBlockedAsPassed)
	} else {
		// true negatives for malicious payloads (IsTruePositive is true)
		// and false positives checks (IsTruePositive is false)
		if (blocked && w.isTruePositive) ||
			(!blocked && !w.isTruePositive) {
			s.db.UpdatePassedTests(info)
		} else {
			s.db.UpdateFailedTests(info)
		}
	}
	return nil
}
