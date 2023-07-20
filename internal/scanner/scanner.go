package scanner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/schollz/progressbar/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"

	"github.com/wallarm/gotestwaf/internal/config"
	"github.com/wallarm/gotestwaf/internal/db"
	"github.com/wallarm/gotestwaf/internal/dnscache"
	"github.com/wallarm/gotestwaf/internal/openapi"
	"github.com/wallarm/gotestwaf/internal/payload/encoder"
	"github.com/wallarm/gotestwaf/internal/payload/placeholder"
)

const (
	preCheckVector        = "<script>alert('union select password from users')</script>"
	wsPreCheckReadTimeout = time.Second * 1
)

type testWork struct {
	setName          string
	caseName         string
	payload          string
	encoder          string
	placeholder      *db.Placeholder
	testType         string
	isTruePositive   bool
	debugHeaderValue string
}

// Scanner allows you to test WAF in various ways with given payloads.
type Scanner struct {
	logger *logrus.Logger
	cfg    *config.Config
	db     *db.DB

	httpClient *HTTPClient
	grpcConn   *GRPCConn
	wsClient   *websocket.Dialer

	requestTemplates openapi.Templates
	router           routers.Router

	enableDebugHeader bool
}

// New creates a new Scanner.
func New(
	logger *logrus.Logger,
	cfg *config.Config,
	db *db.DB,
	dnsResolver *dnscache.Resolver,
	requestTemplates openapi.Templates,
	router routers.Router,
	enableDebugHeader bool,
) (*Scanner, error) {
	httpClient, err := NewHTTPClient(cfg, dnsResolver)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create HTTP client")
	}

	grpcConn, err := NewGRPCConn(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create gRPC client")
	}

	return &Scanner{
		logger:            logger,
		cfg:               cfg,
		db:                db,
		httpClient:        httpClient,
		grpcConn:          grpcConn,
		requestTemplates:  requestTemplates,
		router:            router,
		wsClient:          websocket.DefaultDialer,
		enableDebugHeader: enableDebugHeader,
	}, nil
}

// CheckGRPCAvailability checks if the gRPC server is available at the given URL.
func (s *Scanner) CheckGRPCAvailability(ctx context.Context) {
	s.logger.WithField("status", "started").Info("gRPC pre-check")

	available, err := s.grpcConn.CheckAvailability(ctx)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"status":     "done",
			"connection": "not available",
		}).WithError(err).Infof("gRPC pre-check")
	}
	if available {
		s.logger.WithFields(logrus.Fields{
			"status":     "done",
			"connection": "available",
		}).Info("gRPC pre-check")
	} else {
		s.logger.WithFields(logrus.Fields{
			"status":     "done",
			"connection": "not available",
		}).Info("gRPC pre-check")
	}

	s.db.IsGrpcAvailable = available
}

// WAFBlockCheck checks if WAF exists and blocks malicious requests.
func (s *Scanner) WAFBlockCheck(ctx context.Context) error {
	if !s.cfg.SkipWAFBlockCheck {
		s.logger.WithField("url", s.cfg.URL).Info("WAF pre-check")

		ok, httpStatus, err := s.preCheck(ctx, preCheckVector)
		if err != nil {
			if s.cfg.BlockConnReset && (errors.Is(err, io.EOF) || errors.Is(err, syscall.ECONNRESET)) {
				s.logger.Info("Connection reset, trying benign request to make sure that service is available")
				blockedBenign, httpStatusBenign, errBenign := s.preCheck(ctx, "")
				if !blockedBenign {
					s.logger.Infof("Service is available (HTTP status: %d), WAF resets connections. Consider this behavior as block", httpStatusBenign)
					ok = true
				}
				if errBenign != nil {
					return errors.Wrap(errBenign, "running benign request pre-check")
				}
			} else {
				return errors.Wrap(err, "running WAF pre-check")
			}
		}

		if !ok {
			return errors.Errorf("WAF was not detected. "+
				"Please use the '--blockStatusCodes' or '--blockRegex' flags. Use '--help' for additional info. "+
				"Baseline attack status code: %v", httpStatus)
		}

		s.logger.WithFields(logrus.Fields{
			"status":  "done",
			"blocked": true,
			"code":    httpStatus,
		}).Info("WAF pre-check")
	} else {
		s.logger.WithField("status", "skipped").Info("WAF pre-check")
	}

	return nil
}

// preCheck sends given payload during the pre-check stage.
func (s *Scanner) preCheck(ctx context.Context, payload string) (blocked bool, statusCode int, err error) {
	respMsgHeader, respBody, code, err := s.httpClient.SendPayload(ctx, s.cfg.URL, payload, "URL", "URLParam", nil, "")
	if err != nil {
		return false, 0, err
	}
	blocked, err = s.checkBlocking(respMsgHeader, respBody, code)
	if err != nil {
		return false, 0, err
	}
	return blocked, code, nil
}

// WAFwsBlockCheck checks if WebSocket exists and is protected by WAF.
func (s *Scanner) WAFwsBlockCheck(ctx context.Context) {
	if !s.cfg.SkipWAFBlockCheck {
		s.logger.WithFields(logrus.Fields{
			"status": "started",
			"url":    s.cfg.WebSocketURL,
		}).Info("WebSocket pre-check")

		available, blocked, err := s.wsPreCheck(ctx)
		if !available && err != nil {
			s.logger.WithFields(logrus.Fields{
				"status":     "done",
				"connection": "not available",
			}).WithError(err).Info("WebSocket pre-check")
		}
		if available && blocked {
			s.logger.WithFields(logrus.Fields{
				"status":     "done",
				"connection": "available",
				"blocked":    true,
			}).Info("WebSocket pre-check")
		}
		if available && !blocked {
			s.logger.WithFields(logrus.Fields{
				"status":     "done",
				"connection": "available",
				"blocked":    false,
			}).Info("WebSocket pre-check")
		}
	} else {
		s.logger.WithField("status", "skipped").Info("WebSocket pre-check")
	}
}

// wsPreCheck sends the payload and analyzes response.
func (s *Scanner) wsPreCheck(ctx context.Context) (available, blocked bool, err error) {
	wsClient, _, err := s.wsClient.DialContext(ctx, s.cfg.WebSocketURL, nil)
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

// Run starts a host scan to check WAF security.
func (s *Scanner) Run(ctx context.Context) error {
	gn := s.cfg.Workers
	var wg sync.WaitGroup
	wg.Add(gn)

	defer s.grpcConn.Close()

	rand.Seed(time.Now().UnixNano())

	s.logger.WithField("url", s.cfg.URL).Info("Scanning started")

	start := time.Now()
	defer func() {
		s.logger.WithField("duration", time.Since(start).String()).Info("Scanning finished")
	}()

	testChan := s.produceTests(ctx, gn)

	progressbarOptions := []progressbar.Option{
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
	}

	// disable progress bar output if logging in JSONFormat
	if _, ok := s.logger.Formatter.(*logrus.JSONFormatter); ok {
		progressbarOptions = append(progressbarOptions, progressbar.OptionSetWriter(io.Discard))
	}

	bar := progressbar.NewOptions64(
		int64(s.db.NumberOfTests),
		progressbarOptions...,
	)

	// progressbar doesn't support getting the current value of counter,
	// only the percentage. Because of that we count the number of sent requests
	// separately.
	var requestsCounter uint64

	userSignal := make(chan os.Signal, 1)
	signal.Notify(userSignal, syscall.SIGUSR1)
	defer func() {
		signal.Stop(userSignal)
		close(userSignal)
	}()

	go func() {
		for {
			select {
			case _, ok := <-userSignal:
				if !ok {
					return
				}

				s.logger.
					WithFields(logrus.Fields{
						"sent":  atomic.LoadUint64(&requestsCounter),
						"total": s.db.NumberOfTests,
					}).Info("Testing status")

			case <-ctx.Done():
				return
			}
		}
	}()

	for e := 0; e < gn; e++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case w, ok := <-testChan:
					if !ok {
						return
					}
					time.Sleep(time.Duration(s.cfg.SendDelay+rand.Intn(s.cfg.RandomDelay)) * time.Millisecond)

					if err := s.scanURL(ctx, w); err != nil {
						s.logger.WithError(err).Error("Got an error while scanning")
					}

					// count the number of sent request to show statistics on the SIGUSR1 signal
					atomic.AddUint64(&requestsCounter, 1)
					bar.Add(1)

				case <-ctx.Done():
					return
				}
			}
		}()
	}

	wg.Wait()
	if errors.Is(ctx.Err(), context.Canceled) {
		return ctx.Err()
	}

	return nil
}

// checkBlocking checks the response status-code or request body using
// a regular expression to determine if the request has been blocked.
func (s *Scanner) checkBlocking(responseMsgHeader, body string, statusCode int) (bool, error) {
	if s.cfg.BlockRegex != "" {
		response := body
		if responseMsgHeader != "" {
			response = responseMsgHeader + body
		}

		if response != "" {
			m, _ := regexp.MatchString(s.cfg.BlockRegex, response)

			return m, nil
		}
	}

	for _, code := range s.cfg.BlockStatusCodes {
		if statusCode == code {
			return true, nil
		}
	}

	return false, nil
}

// checkPass checks the response status-code or request body using
// a regular expression to determine if the request has been passed.
func (s *Scanner) checkPass(responseMsgHeader, body string, statusCode int) (bool, error) {
	if s.cfg.PassRegex != "" {
		response := body
		if responseMsgHeader != "" {
			response = responseMsgHeader + body
		}

		if response != "" {
			m, _ := regexp.MatchString(s.cfg.PassRegex, response)

			return m, nil
		}
	}

	for _, code := range s.cfg.PassStatusCodes {
		if statusCode == code {
			return true, nil
		}
	}

	return false, nil
}

// produceTests generates all combinations of payload, encoder, and placeholder
// for n goroutines.
func (s *Scanner) produceTests(ctx context.Context, n int) <-chan *testWork {
	testChan := make(chan *testWork, n)
	testCases := s.db.GetTestCases()

	go func() {
		defer close(testChan)

		var debugHeaderValue string

		hash := sha256.New()

		for _, testCase := range testCases {
			for _, payload := range testCase.Payloads {
				for _, encoder := range testCase.Encoders {
					for _, placeholder := range testCase.Placeholders {
						if s.enableDebugHeader {
							hash.Reset()

							hash.Write([]byte(testCase.Set))
							hash.Write([]byte(testCase.Name))
							hash.Write([]byte(placeholder.Name))
							hash.Write([]byte(encoder))
							hash.Write([]byte(payload))

							debugHeaderValue = hex.EncodeToString(hash.Sum(nil))
						} else {
							debugHeaderValue = ""
						}

						wrk := &testWork{
							setName:          testCase.Set,
							caseName:         testCase.Name,
							payload:          payload,
							encoder:          encoder,
							placeholder:      placeholder,
							testType:         testCase.Type,
							isTruePositive:   testCase.IsTruePositive,
							debugHeaderValue: debugHeaderValue,
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

// scanURL scans the host with the given combination of payload, encoder and
// placeholder.
func (s *Scanner) scanURL(ctx context.Context, w *testWork) error {
	var (
		respHeaders   http.Header
		respMsgHeader string
		respBody      string
		statusCode    int
		err           error
	)

	if w.placeholder.Name == placeholder.DefaultGRPC.GetName() {
		if !s.grpcConn.IsAvailable() {
			return nil
		}

		newCtx := ctx
		if w.debugHeaderValue != "" {
			newCtx = metadata.AppendToOutgoingContext(ctx, GTWDebugHeader, w.debugHeaderValue)
		}

		respBody, statusCode, err = s.grpcConn.Send(newCtx, w.encoder, w.payload)

		_, _, _, _, err = s.updateDB(ctx, w, nil, nil, nil, nil, nil,
			statusCode, nil, "", respBody, err, "", true)

		return err
	}

	if s.requestTemplates == nil {
		respMsgHeader, respBody, statusCode, err = s.httpClient.SendPayload(ctx, s.cfg.URL, w.payload, w.encoder, w.placeholder.Name, w.placeholder.Config, w.debugHeaderValue)

		_, _, _, _, err = s.updateDB(ctx, w, nil, nil, nil, nil, nil,
			statusCode, nil, respMsgHeader, respBody, err, "", false)

		return err
	}

	templates := s.requestTemplates[w.placeholder.Name]

	encodedPayload, err := encoder.Apply(w.encoder, w.payload)
	if err != nil {
		return errors.Wrap(err, "encoding payload")
	}

	var passedTest *db.Info
	var blockedTest *db.Info
	var unresolvedTest *db.Info
	var failedTest *db.Info
	var additionalInfo string

	for _, template := range templates {
		req, err := template.CreateRequest(ctx, w.placeholder.Name, encodedPayload)
		if err != nil {
			return errors.Wrap(err, "create request from template")
		}

		respHeaders, respMsgHeader, respBody, statusCode, err = s.httpClient.SendRequest(req, w.debugHeaderValue)

		additionalInfo = fmt.Sprintf("%s %s", template.Method, template.Path)

		passedTest, blockedTest, unresolvedTest, failedTest, err =
			s.updateDB(ctx, w, passedTest, blockedTest, unresolvedTest, failedTest,
				req, statusCode, respHeaders, respMsgHeader, respBody, err, additionalInfo, false)

		s.db.AddToScannedPaths(template.Method, template.Path)

		if err != nil {
			return err
		}
	}

	return nil
}

// updateDB updates the success of a query in the database.
func (s *Scanner) updateDB(
	ctx context.Context,
	w *testWork,
	passedTest *db.Info,
	blockedTest *db.Info,
	unresolvedTest *db.Info,
	failedTest *db.Info,
	req *http.Request,
	respStatusCode int,
	respHeaders http.Header,
	respMsgHeader string,
	respBody string,
	sendErr error,
	additionalInfo string,
	isGRPC bool,
) (
	updPassedTest *db.Info,
	updBlockedTest *db.Info,
	updUnresolvedTest *db.Info,
	updFailedTest *db.Info,
	err error,
) {
	updPassedTest = passedTest
	updBlockedTest = blockedTest
	updUnresolvedTest = unresolvedTest
	updFailedTest = failedTest

	info := w.toInfo(respStatusCode)

	var blockedByReset bool
	if sendErr != nil {
		if errors.Is(sendErr, io.EOF) || errors.Is(sendErr, syscall.ECONNRESET) {
			if s.cfg.BlockConnReset {
				blockedByReset = true
			} else {
				if updUnresolvedTest == nil {
					updUnresolvedTest = info
					s.db.UpdateNaTests(updUnresolvedTest, s.cfg.IgnoreUnresolved, s.cfg.NonBlockedAsPassed, w.isTruePositive)
				}
				if len(additionalInfo) != 0 {
					unresolvedTest.AdditionalInfo = append(unresolvedTest.AdditionalInfo, additionalInfo)
				}

				return
			}
		} else {
			if updFailedTest == nil {
				updFailedTest = info
				s.db.UpdateFailedTests(updFailedTest)
			}
			if len(additionalInfo) != 0 {
				updFailedTest.AdditionalInfo = append(updFailedTest.AdditionalInfo, sendErr.Error())
			}

			s.logger.WithError(sendErr).Error("send request failed")

			return
		}
	}

	var blocked, passed bool
	if blockedByReset {
		blocked = true
	} else {
		blocked, err = s.checkBlocking(respMsgHeader, respBody, respStatusCode)
		if err != nil {
			return nil, nil, nil, nil,
				errors.Wrap(err, "failed to check blocking")
		}

		passed, err = s.checkPass(respMsgHeader, respBody, respStatusCode)
		if err != nil {
			return nil, nil, nil, nil,
				errors.Wrap(err, "failed to check passed or not")
		}
	}

	if s.requestTemplates != nil && !isGRPC {
		route, pathParams, routeErr := s.router.FindRoute(req)
		if routeErr != nil {
			// split Method and url template
			additionalInfoParts := strings.Split(additionalInfo, " ")
			if len(additionalInfoParts) < 2 {
				return nil, nil, nil, nil,
					errors.Wrap(routeErr, "couldn't find request route")
			}

			req.URL.Path = additionalInfoParts[1]
			route, pathParams, routeErr = s.router.FindRoute(req)
			if routeErr != nil {
				return nil, nil, nil, nil,
					errors.Wrap(routeErr, "couldn't find request route")
			}
		}

		inputReuqestValidation := &openapi3filter.RequestValidationInput{
			Request:     req,
			PathParams:  pathParams,
			QueryParams: req.URL.Query(),
			Route:       route,
		}

		responseValidationInput := &openapi3filter.ResponseValidationInput{
			RequestValidationInput: inputReuqestValidation,
			Status:                 respStatusCode,
			Header:                 respHeaders,
			Body:                   io.NopCloser(strings.NewReader(respBody)),
			Options: &openapi3filter.Options{
				IncludeResponseStatus: true,
			},
		}

		if validationErr := openapi3filter.ValidateResponse(ctx, responseValidationInput); validationErr == nil && !blocked {
			if updPassedTest == nil {
				updPassedTest = info
				s.db.UpdatePassedTests(updPassedTest)
			}
			if len(additionalInfo) != 0 {
				updPassedTest.AdditionalInfo = append(updPassedTest.AdditionalInfo, additionalInfo)
			}
		} else {
			if updBlockedTest == nil {
				updBlockedTest = info
				s.db.UpdateBlockedTests(updBlockedTest)
			}
			if len(additionalInfo) != 0 {
				updBlockedTest.AdditionalInfo = append(updBlockedTest.AdditionalInfo, additionalInfo)
			}
		}

		return
	}

	if (blocked && passed) || (!blocked && !passed) {
		if updUnresolvedTest == nil {
			updUnresolvedTest = info
			s.db.UpdateNaTests(updUnresolvedTest, s.cfg.IgnoreUnresolved, s.cfg.NonBlockedAsPassed, w.isTruePositive)
		}
		if len(additionalInfo) != 0 {
			unresolvedTest.AdditionalInfo = append(unresolvedTest.AdditionalInfo, additionalInfo)
		}
	} else {
		if blocked {
			if updBlockedTest == nil {
				updBlockedTest = info
				s.db.UpdateBlockedTests(updBlockedTest)
			}
			if len(additionalInfo) != 0 {
				updBlockedTest.AdditionalInfo = append(updBlockedTest.AdditionalInfo, additionalInfo)
			}
		} else {
			if updPassedTest == nil {
				updPassedTest = info
				s.db.UpdatePassedTests(updPassedTest)
			}
			if len(additionalInfo) != 0 {
				updPassedTest.AdditionalInfo = append(updPassedTest.AdditionalInfo, additionalInfo)
			}
		}
	}

	return
}

func (w *testWork) toInfo(respStatusCode int) *db.Info {
	return &db.Info{
		Set:                w.setName,
		Case:               w.caseName,
		Payload:            w.payload,
		Encoder:            w.encoder,
		Placeholder:        w.placeholder.Name,
		ResponseStatusCode: respStatusCode,
		Type:               w.testType,
	}
}
