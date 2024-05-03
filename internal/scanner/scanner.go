package scanner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/wallarm/gotestwaf/internal/scanner/types"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/schollz/progressbar/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"

	"github.com/wallarm/gotestwaf/internal/config"
	"github.com/wallarm/gotestwaf/internal/db"
	"github.com/wallarm/gotestwaf/internal/helpers"
	"github.com/wallarm/gotestwaf/internal/openapi"
	p "github.com/wallarm/gotestwaf/internal/payload"
	"github.com/wallarm/gotestwaf/internal/payload/encoder"
	"github.com/wallarm/gotestwaf/internal/payload/placeholder"
	"github.com/wallarm/gotestwaf/internal/scanner/clients"
	"github.com/wallarm/gotestwaf/internal/scanner/waf_detector/detectors"
	"github.com/wallarm/gotestwaf/pkg/dnscache"
)

const (
	preCheckVector        = "<script>alert('union select password from users')</script>"
	wsPreCheckReadTimeout = time.Second * 1
)

var jsChallengeErrorMsgs = []string{
	"Enable JavaScript and cookies to continue",
}

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

	httpClient *clients.GoHTTPClient
	grpcConn   *clients.GRPCConn
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
	httpClient, err := clients.NewGoHTTPClient(cfg, dnsResolver)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create HTTP client")
	}

	grpcConn, err := clients.NewGRPCConn(cfg)
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

func (s *Scanner) CheckIfJavaScriptRequired(ctx context.Context) (bool, error) {
	fullUrl, _ := url.Parse(s.cfg.URL)
	reducedUrl := helpers.GetTargetURLStr(fullUrl)

	rawRequest, err := http.NewRequest("GET", reducedUrl, nil)
	if err != nil {
		return false, err
	}
	rawRequest = rawRequest.WithContext(ctx)

	resp, err := s.httpClient.SendRequest(ctx, &types.GoHTTPRequest{Req: rawRequest})
	if err != nil {
		return false, err
	}

	body := string(resp.GetContent())

	for i := range jsChallengeErrorMsgs {
		if strings.Contains(body, jsChallengeErrorMsgs[i]) {
			return true, nil
		}
	}

	return false, nil
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
	pl := &p.PayloadInfo{
		Payload:         payload,
		EncoderName:     "URL",
		PlaceholderName: "URLParam",
	}

	resp, err := s.httpClient.SendPayload(ctx, s.cfg.URL, pl)
	if err != nil {
		return false, 0, err
	}

	blocked, _, err = s.checkBlockedOrPassed(resp)
	if err != nil {
		return false, 0, err
	}

	return blocked, resp.GetStatusCode(), nil
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

// checkBlockedOrPassed checks the response status-code or request body using
// a regular expression to determine if the request has been blocked or passed.
func (s *Scanner) checkBlockedOrPassed(
	resp types.Response,
) (blocked, passed bool, err error) {
	if s.cfg.CheckBlockFunc != nil {
		if s.cfg.CheckBlockFunc(&detectors.Responses{RespToAttack: resp}) {
			return true, false, nil
		}
	}

	var headers []string
	for header, value := range resp.GetHeaders() {
		headers = append(headers, fmt.Sprintf("%s: %s", header, value[0]))
	}

	body := resp.GetContent()
	statusCode := resp.GetStatusCode()

	if s.cfg.BlockRegex != "" {
		for _, header := range headers {
			matched, _ := regexp.MatchString(s.cfg.BlockRegex, header)
			if matched {
				blocked = true
			}
		}

		if blocked == false && body != nil && len(body) > 0 {
			matched, _ := regexp.Match(s.cfg.BlockRegex, body)

			blocked = matched
		}
	}

	if s.cfg.PassRegex != "" {
		for _, header := range headers {
			matched, _ := regexp.MatchString(s.cfg.PassRegex, header)
			if matched {
				passed = true
			}
		}

		if passed == false && body != nil && len(body) > 0 {
			matched, _ := regexp.Match(s.cfg.PassRegex, body)

			passed = matched
		}
	}

	for _, code := range s.cfg.BlockStatusCodes {
		if statusCode == code {
			blocked = true
		}
	}

	for _, code := range s.cfg.PassStatusCodes {
		if statusCode == code {
			passed = true
		}
	}

	return
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
		resp       types.Response
		respBody   string
		statusCode int
		err        error
	)

	if w.placeholder.Name == placeholder.DefaultGRPC.GetName() {
		if !s.grpcConn.IsAvailable() {
			return nil
		}

		newCtx := ctx
		if w.debugHeaderValue != "" {
			newCtx = metadata.AppendToOutgoingContext(ctx, clients.GTWDebugHeader, w.debugHeaderValue)
		}

		respBody, statusCode, err = s.grpcConn.Send(newCtx, w.encoder, w.payload)

		resp = &types.ResponseMeta{
			StatusCode: statusCode,
			Content:    []byte(respBody),
		}

		_, _, _, _, err = s.updateDB(ctx, w, nil, nil, nil, nil, nil, resp, err, "", true)

		return err
	}

	if s.requestTemplates == nil {
		pl := &p.PayloadInfo{
			Payload:           w.payload,
			EncoderName:       w.encoder,
			PlaceholderName:   w.placeholder.Name,
			PlaceholderConfig: w.placeholder.Config,
			DebugHeaderValue:  w.debugHeaderValue,
		}

		resp, err = s.httpClient.SendPayload(ctx, s.cfg.URL, pl)

		_, _, _, _, err = s.updateDB(ctx, w, nil, nil, nil, nil, nil, resp, err, "", false)

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

		resp, err = s.httpClient.SendRequest(ctx, &types.GoHTTPRequest{Req: req})

		additionalInfo = fmt.Sprintf("%s %s", template.Method, template.Path)

		passedTest, blockedTest, unresolvedTest, failedTest, err =
			s.updateDB(ctx, w, passedTest, blockedTest, unresolvedTest, failedTest,
				req, resp, err, additionalInfo, false)

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
	resp types.Response,
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

	info := w.toInfo(resp.GetStatusCode())

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
		blocked, passed, err = s.checkBlockedOrPassed(resp)
		if err != nil {
			return nil, nil, nil, nil,
				errors.Wrap(err, "failed to check blocking")
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
			Status:                 resp.GetStatusCode(),
			Header:                 resp.GetHeaders(),
			Body:                   io.NopCloser(bytes.NewReader(resp.GetContent())),
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
