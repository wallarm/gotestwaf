package scanner

import (
	"context"
	"log"
	"math/rand"
	"regexp"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/wallarm/gotestwaf/internal/data/config"
	"github.com/wallarm/gotestwaf/internal/data/test"
	"github.com/wallarm/gotestwaf/internal/payload/encoder"
)

const preCheckVector = "<script>alert('union select password from users')</script>"

var i int

type testWork struct {
	set         string
	name        string
	payload     string
	encoder     string
	placeholder string
	tp          bool
}

type Scanner struct {
	logger     *log.Logger
	cfg        *config.Config
	db         *test.DB
	httpClient *HTTPClient
}

func New(db *test.DB, logger *log.Logger, cfg *config.Config) *Scanner {
	return &Scanner{
		db:         db,
		logger:     logger,
		cfg:        cfg,
		httpClient: NewHTTPClient(cfg),
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

func (s *Scanner) PreCheck(url string) (bool, int, error) {
	encoder.InitEncoders()
	body, code, err := s.httpClient.Send(context.Background(), url, "URLParam", "URL", preCheckVector)
	if err != nil {
		return false, 0, err
	}
	ok, err := s.CheckBlocking(body, code)
	if err != nil {
		return false, 0, err
	}
	return ok, code, nil
}

func (s *Scanner) Run(ctx context.Context, url string) error {
	encoder.InitEncoders()
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

	workChan := make(chan testWork, gn)

	for e := 0; e < gn; e++ {
		go func(ctx context.Context) {
			defer wg.Done()
			for {
				select {
				case w, ok := <-workChan:
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

	testCases := s.db.GetTests()

	go func() {
		defer close(workChan)
		for _, t := range testCases {
			for _, payload := range t.Payloads {
				for _, e := range t.Encoders {
					for _, placeholder := range t.Placeholders {
						wrk := testWork{t.Set, t.Name, payload, e, placeholder, t.Type}
						select {
						case workChan <- wrk:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}
	}()
	wg.Wait()
	if err := errors.Cause(ctx.Err()); err == context.Canceled {
		return ctx.Err()
	}
	return nil
}

func (s *Scanner) scanURL(ctx context.Context, url string, w testWork) error {
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
	t := &test.Test{
		TestSet:     w.set,
		TestCase:    w.name,
		Payload:     w.payload,
		Encoder:     w.encoder,
		Placeholder: w.placeholder,
		StatusCode:  statusCode,
	}
	if (blocked && passed) || (!blocked && !passed) {
		s.db.UpdateNaTests(t, s.cfg.NonBlockedAsPassed)
	} else {
		// true positives
		if (blocked && w.tp) ||
			// true negatives for malicious payloads (Type is true)
			// and false positives checks (Type is false)
			(!blocked && !w.tp) {
			s.db.UpdatePassedTests(t)
		} else {
			s.db.UpdateFailedTests(t)
		}
	}
	return nil
}
