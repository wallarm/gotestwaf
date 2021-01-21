package scanner

import (
	"log"
	"math/rand"
	"regexp"
	"sync"
	"time"

	"github.com/wallarm/gotestwaf/internal/data/config"
	"github.com/wallarm/gotestwaf/internal/data/test"
	"github.com/wallarm/gotestwaf/internal/payload/encoder"
	"github.com/wallarm/gotestwaf/internal/platform"
)

const preCheckVector = "<script>alert('union select password from users')</script>"

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
	body, code, err := s.httpClient.Send(url, "URLParam", "URL", preCheckVector)
	if err != nil {
		return false, 0, err
	}
	ok, err := s.CheckBlocking(body, code)
	if err != nil {
		return false, 0, err
	}
	return ok, code, nil
}

func (s *Scanner) Run(url string) error {
	encoder.InitEncoders()
	gn := s.cfg.Workers

	var wg sync.WaitGroup
	wg.Add(gn)

	workChan := make(chan testWork, gn)

	rand.Seed(time.Now().UnixNano())

	bar := platform.NewProgressBar()

	s.logger.Println("Scanning started")
	defer s.logger.Println("Scanning finished")

	start := time.Now()
	defer func() {
		s.logger.Println("Scanning Time: ", time.Since(start))
	}()

	for e := 0; e < gn; e++ {
		go func() {
			defer wg.Done()
			for w := range workChan {
				time.Sleep(time.Duration(s.cfg.SendDelay+rand.Intn(s.cfg.RandomDelay)) * time.Millisecond)
				body, statusCode, _ := s.httpClient.Send(url, w.placeholder, w.encoder, w.payload)

				blocked, err := s.CheckBlocking(body, statusCode)
				if err != nil {
					s.logger.Println("failed to check blocking:", err)
				}
				passed, err := s.CheckPass(body, statusCode)
				if err != nil {
					s.logger.Println("failed to check passed or not:", err)
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
				_ = bar.Add(1)
			}
		}()
	}

	testCases := s.db.GetTests()

	for _, t := range testCases {
		for _, payload := range t.Payloads {
			for _, e := range t.Encoders {
				for _, placeholder := range t.Placeholders {
					wrk := testWork{
						t.Set,
						t.Name,
						payload,
						e,
						placeholder,
						t.Type,
					}
					workChan <- wrk
				}
			}
		}
	}

	close(workChan)
	wg.Wait()
	_ = bar.Finish()

	return nil
}
