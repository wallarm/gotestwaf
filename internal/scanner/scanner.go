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

	"github.com/wallarm/gotestwaf/internal/scanner/clients/graphql"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/pkg/errors"
	"github.com/schollz/progressbar/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"

	"github.com/wallarm/gotestwaf/internal/config"
	"github.com/wallarm/gotestwaf/internal/db"
	dns_cache "github.com/wallarm/gotestwaf/internal/dnscache"
	"github.com/wallarm/gotestwaf/internal/helpers"
	"github.com/wallarm/gotestwaf/internal/openapi"
	p "github.com/wallarm/gotestwaf/internal/payload"
	"github.com/wallarm/gotestwaf/internal/payload/encoder"
	"github.com/wallarm/gotestwaf/internal/payload/placeholder"
	"github.com/wallarm/gotestwaf/internal/scanner/clients"
	"github.com/wallarm/gotestwaf/internal/scanner/clients/chrome"
	"github.com/wallarm/gotestwaf/internal/scanner/clients/gohttp"
	"github.com/wallarm/gotestwaf/internal/scanner/clients/grpc"
	"github.com/wallarm/gotestwaf/internal/scanner/types"
	"github.com/wallarm/gotestwaf/internal/scanner/waf_detector/detectors"
	"github.com/wallarm/gotestwaf/pkg/dnscache"
)

const (
	preCheckVector = "<script>alert('union select password from users')</script>"
)

var jsChallengeErrorMsgs = []string{
	"Enable JavaScript and cookies to continue",
}

type payloadConfig struct {
	payload     string
	encoder     string
	placeholder *db.Placeholder

	setName        string
	caseName       string
	testType       string
	isTruePositive bool

	debugHeaderValue string
}

type testStatus struct {
	passedTest     *db.Info
	blockedTest    *db.Info
	unresolvedTest *db.Info
	failedTest     *db.Info
}

// Scanner allows you to test WAF in various ways with given payloads.
type Scanner struct {
	logger *logrus.Logger
	cfg    *config.Config
	db     *db.DB

	httpClient    clients.HTTPClient
	grpcConn      clients.GRPCClient
	graphqlClient clients.GraphQLClient

	requestTemplates openapi.Templates
	router           routers.Router

	enableDebugHeader bool
}

// New creates a new Scanner.
func New(
	logger *logrus.Logger,
	cfg *config.Config,
	db *db.DB,
	requestTemplates openapi.Templates,
	router routers.Router,
	enableDebugHeader bool,
) (*Scanner, error) {
	var (
		httpClient clients.HTTPClient
		dnsCache   *dnscache.Resolver
		err        error
	)

	dnsCache, err = dns_cache.NewDNSCache(logger)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create DNS cache")
	}

	if cfg.HTTPClient == "chrome" {
		httpClient, err = chrome.NewClient(cfg)
	} else {
		httpClient, err = gohttp.NewClient(cfg, dnsCache)
	}
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create HTTP client")
	}

	grpcConn, err := grpc.NewClient(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create gRPC client")
	}

	graphqlClient, err := graphql.NewClient(cfg, dnsCache)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create GraphQL client")
	}

	return &Scanner{
		logger:            logger,
		cfg:               cfg,
		db:                db,
		httpClient:        httpClient,
		grpcConn:          grpcConn,
		graphqlClient:     graphqlClient,
		requestTemplates:  requestTemplates,
		router:            router,
		enableDebugHeader: enableDebugHeader,
	}, nil
}

func (s *Scanner) CheckIfJavaScriptRequired(ctx context.Context) (bool, error) {
	fullUrl, _ := url.Parse(s.cfg.URL)
	reducedUrl := helpers.GetTargetURLStr(fullUrl)

	rawRequest, err := http.NewRequest(http.MethodGet, reducedUrl, nil)
	if err != nil {
		return false, err
	}
	rawRequest = rawRequest.WithContext(ctx)

	cfgFixed := *s.cfg
	cfgFixed.FollowCookies = true
	cfgFixed.RenewSession = true

	client, err := gohttp.NewClient(&cfgFixed, nil)
	if err != nil {
		return false, err
	}

	resp, err := client.SendRequest(ctx, &types.GoHTTPRequest{Req: rawRequest})
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

	connection := "not available"
	if available {
		connection = "available"
	}

	s.logger.WithFields(logrus.Fields{
		"status":     "done",
		"connection": connection,
	}).Info("gRPC pre-check")

	s.db.IsGrpcAvailable = available
}

// CheckGraphQLAvailability checks if the GraphQL is available at the given URL.
func (s *Scanner) CheckGraphQLAvailability(ctx context.Context) {
	s.logger.WithField("status", "started").Info("GraphQL pre-check")

	available, err := s.graphqlClient.CheckAvailability(ctx)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"status":     "done",
			"connection": "not available",
		}).WithError(err).Infof("GraphQL pre-check")
	}

	s.db.IsGrpcAvailable = available
	connection := "not available"
	if available {
		connection = "available"
	}

	s.logger.WithFields(logrus.Fields{
		"status":     "done",
		"connection": connection,
	}).Info("GraphQL pre-check")

	s.db.IsGraphQLAvailable = available
}

// WAFBlockCheck checks if WAF exists and blocks malicious requests.
func (s *Scanner) WAFBlockCheck(ctx context.Context) error {
	s.logger.WithField("url", s.cfg.URL).Info("WAF pre-check")

	ok, httpStatus, err := s.preCheck(ctx, preCheckVector)
	if err != nil {
		if s.cfg.BlockConnReset && (errors.Is(err, io.EOF) || errors.Is(err, syscall.ECONNRESET)) {
			s.logger.Info("Connection reset, trying benign request to make sure that service is available")

			blockedBenign, httpStatusBenign, errBenign := s.preCheck(ctx, "")
			if !blockedBenign {
				s.logger.Infof(
					"Service is available (HTTP status: %d), "+
						"WAF resets connections. Consider this behavior as block",
					httpStatusBenign,
				)
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
			"Please use the '--blockStatusCodes' or '--blockRegex' flags. "+
			"Use '--help' for additional info. "+
			"Baseline attack status code: %v",
			httpStatus,
		)
	}

	s.logger.WithFields(logrus.Fields{
		"status":  "done",
		"blocked": true,
		"code":    httpStatus,
	}).Info("WAF pre-check")

	return nil
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

	payloadChan := s.produceTests(ctx, gn)

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
				case pc, ok := <-payloadChan:
					if !ok {
						return
					}
					time.Sleep(time.Duration(s.cfg.SendDelay+rand.Intn(s.cfg.RandomDelay)) * time.Millisecond)

					if err := s.sendPayload(ctx, pc); err != nil {
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
func (s *Scanner) produceTests(ctx context.Context, n int) <-chan *payloadConfig {
	payloadChan := make(chan *payloadConfig, n)
	testCases := s.db.GetTestCases()

	go func() {
		defer close(payloadChan)

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

						wrk := &payloadConfig{
							payload:     payload,
							encoder:     encoder,
							placeholder: placeholder,

							setName:        testCase.Set,
							caseName:       testCase.Name,
							testType:       testCase.Type,
							isTruePositive: testCase.IsTruePositive,

							debugHeaderValue: debugHeaderValue,
						}

						select {
						case payloadChan <- wrk:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}
	}()

	return payloadChan
}

// sendPayload sends a payload based on the provided configuration
func (s *Scanner) sendPayload(ctx context.Context, pc *payloadConfig) error {
	var err error

	if pc.placeholder.Name == placeholder.DefaultGRPC.GetName() {
		return s.sendGrpcRequest(ctx, pc)
	}

	if pc.placeholder.Name == placeholder.DefaultGraphQL.GetName() {
		return s.sendGraphQLRequest(ctx, pc)
	}

	if s.requestTemplates != nil {
		err = s.sendOpenAPIRequests(ctx, pc)
		if err != nil {
			return err
		}
	}

	err = s.sendRequest(ctx, pc)
	if err != nil {
		return err
	}

	return nil
}

// sendGrpcRequest sends a gRPC request with the provided payload configuration.
// It checks the availability of the gRPC connection before sending request.
func (s *Scanner) sendGrpcRequest(ctx context.Context, pc *payloadConfig) error {
	if !s.grpcConn.IsAvailable() {
		return nil
	}

	newCtx := ctx
	if pc.debugHeaderValue != "" {
		newCtx = metadata.AppendToOutgoingContext(ctx, clients.GTWDebugHeader, pc.debugHeaderValue)
	}

	pl := &p.PayloadInfo{
		Payload:     pc.payload,
		EncoderName: pc.encoder,
	}

	resp, err := s.grpcConn.SendPayload(newCtx, pl)

	err = s.updateDB(ctx, pc, &testStatus{}, nil, resp, err, "", true)

	return err
}

// sendGraphQLRequest sends a GraphQL request with the provided payload configuration.
// It checks the availability of the GraphQL endpoint before sending request.
func (s *Scanner) sendGraphQLRequest(ctx context.Context, pc *payloadConfig) error {
	if !s.graphqlClient.IsAvailable() {
		return nil
	}

	var (
		resp types.Response
		err  error
	)

	pl := &p.PayloadInfo{
		Payload:           pc.payload,
		EncoderName:       pc.encoder,
		PlaceholderName:   pc.placeholder.Name,
		PlaceholderConfig: pc.placeholder.Config,
		DebugHeaderValue:  pc.debugHeaderValue,
	}

	resp, err = s.graphqlClient.SendPayload(ctx, pl)
	if err == nil && resp != nil {
		err = resp.GetError()
	}

	err = s.updateDB(ctx, pc, &testStatus{}, nil, resp, err, "", false)

	return err
}

// sendRequest sends an HTTP request with the provided payload configuration.
func (s *Scanner) sendRequest(ctx context.Context, pc *payloadConfig) error {
	var (
		resp types.Response
		err  error
	)

	pl := &p.PayloadInfo{
		Payload:           pc.payload,
		EncoderName:       pc.encoder,
		PlaceholderName:   pc.placeholder.Name,
		PlaceholderConfig: pc.placeholder.Config,
		DebugHeaderValue:  pc.debugHeaderValue,
	}

	resp, err = s.httpClient.SendPayload(ctx, s.cfg.URL, pl)
	if err == nil && resp != nil {
		err = resp.GetError()
	}

	err = s.updateDB(ctx, pc, &testStatus{}, nil, resp, err, "", false)

	return err
}

// sendOpenAPIRequests sends multiple HTTP requests based on OpenAPI request templates.
// It iterates over the templates, encodes the payload, creates requests from templates,
// and sends them using the HTTP client.
func (s *Scanner) sendOpenAPIRequests(ctx context.Context, pc *payloadConfig) error {
	var (
		r    *http.Request
		req  types.Request
		resp types.Response
		err  error
	)

	templates := s.requestTemplates[pc.placeholder.Name]

	encodedPayload, err := encoder.Apply(pc.encoder, pc.payload)
	if err != nil {
		return errors.Wrap(err, "encoding payload")
	}

	var additionalInfo string
	ts := &testStatus{}

	for _, template := range templates {
		r, err = template.CreateRequest(ctx, pc.placeholder.Name, encodedPayload)
		if err != nil {
			return errors.Wrap(err, "create request from template")
		}

		req = &types.GoHTTPRequest{Req: r}
		resp, err = s.httpClient.SendRequest(ctx, req)

		additionalInfo = fmt.Sprintf("%s %s", template.Method, template.Path)

		err = s.updateDB(ctx, pc, ts, req, resp, err, additionalInfo, false)

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
	payloadConfig *payloadConfig,
	ts *testStatus,
	req types.Request,
	resp types.Response,
	sendErr error,
	additionalInfo string,
	isGRPC bool,
) (err error) {
	info := payloadConfig.toInfo(resp)

	var blockedByReset bool
	if sendErr != nil {
		if errors.Is(sendErr, io.EOF) || errors.Is(sendErr, syscall.ECONNRESET) {
			if s.cfg.BlockConnReset {
				blockedByReset = true
			} else {
				if ts.unresolvedTest == nil {
					ts.unresolvedTest = info
					s.db.UpdateNaTests(ts.unresolvedTest, s.cfg.IgnoreUnresolved, s.cfg.NonBlockedAsPassed, payloadConfig.isTruePositive)
				}
				if len(additionalInfo) != 0 {
					ts.unresolvedTest.AdditionalInfo = append(ts.unresolvedTest.AdditionalInfo, additionalInfo)
				}

				return
			}
		} else {
			if ts.failedTest == nil {
				ts.failedTest = info
				s.db.UpdateFailedTests(ts.failedTest)
			}
			if len(additionalInfo) != 0 {
				ts.failedTest.AdditionalInfo = append(ts.failedTest.AdditionalInfo, sendErr.Error())
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
			return errors.Wrap(err, "failed to check blocking")
		}
	}

	if s.requestTemplates != nil && !isGRPC {
		r, ok := req.(*types.GoHTTPRequest)
		if !ok {
			return errors.Errorf("bad request type: %T, expected %T", req, &types.GoHTTPRequest{})
		}

		route, pathParams, routeErr := s.router.FindRoute(r.Req)
		if routeErr != nil {
			// split Method and url template
			additionalInfoParts := strings.Split(additionalInfo, " ")
			if len(additionalInfoParts) < 2 {
				return errors.Wrap(routeErr, "couldn't find request route")
			}

			r.Req.URL.Path = additionalInfoParts[1]
			route, pathParams, routeErr = s.router.FindRoute(r.Req)
			if routeErr != nil {
				return errors.Wrap(routeErr, "couldn't find request route")
			}
		}

		inputReuqestValidation := &openapi3filter.RequestValidationInput{
			Request:     r.Req,
			PathParams:  pathParams,
			QueryParams: r.Req.URL.Query(),
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
			if ts.passedTest == nil {
				ts.passedTest = info
				s.db.UpdatePassedTests(ts.passedTest)
			}
			if len(additionalInfo) != 0 {
				ts.passedTest.AdditionalInfo = append(ts.passedTest.AdditionalInfo, additionalInfo)
			}
		} else {
			if ts.blockedTest == nil {
				ts.blockedTest = info
				s.db.UpdateBlockedTests(ts.blockedTest)
			}
			if len(additionalInfo) != 0 {
				ts.blockedTest.AdditionalInfo = append(ts.blockedTest.AdditionalInfo, additionalInfo)
			}
		}

		return
	}

	if (blocked && passed) || (!blocked && !passed) {
		if ts.unresolvedTest == nil {
			ts.unresolvedTest = info
			s.db.UpdateNaTests(ts.unresolvedTest, s.cfg.IgnoreUnresolved, s.cfg.NonBlockedAsPassed, payloadConfig.isTruePositive)
		}
		if len(additionalInfo) != 0 {
			ts.unresolvedTest.AdditionalInfo = append(ts.unresolvedTest.AdditionalInfo, additionalInfo)
		}
	} else {
		if blocked {
			if ts.blockedTest == nil {
				ts.blockedTest = info
				s.db.UpdateBlockedTests(ts.blockedTest)
			}
			if len(additionalInfo) != 0 {
				ts.blockedTest.AdditionalInfo = append(ts.blockedTest.AdditionalInfo, additionalInfo)
			}
		} else {
			if ts.passedTest == nil {
				ts.passedTest = info
				s.db.UpdatePassedTests(ts.passedTest)
			}
			if len(additionalInfo) != 0 {
				ts.passedTest.AdditionalInfo = append(ts.passedTest.AdditionalInfo, additionalInfo)
			}
		}
	}

	return
}

func (pc *payloadConfig) toInfo(resp types.Response) *db.Info {
	info := &db.Info{
		Set:         pc.setName,
		Case:        pc.caseName,
		Payload:     pc.payload,
		Encoder:     pc.encoder,
		Placeholder: pc.placeholder.Name,
		Type:        pc.testType,
	}

	if resp != nil {
		info.ResponseStatusCode = resp.GetStatusCode()
	}

	return info
}
