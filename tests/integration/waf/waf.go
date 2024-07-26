package waf

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/wallarm/gotestwaf/internal/scanner/clients"

	"google.golang.org/grpc"

	pb "github.com/wallarm/gotestwaf/internal/payload/placeholder/grpc"
	"github.com/wallarm/gotestwaf/tests/integration/config"
)

var _ http.Handler = &WAF{}

type WAF struct {
	errChan    chan<- error
	casesMap   *config.TestCasesMap
	httpServer *http.Server
	grpcServer *grpc.Server

	httpPort int
	grpcPort int
}

func New(errChan chan<- error, casesMap *config.TestCasesMap, httpPort int, grpcPort int) *WAF {
	waf := &WAF{
		errChan:  errChan,
		casesMap: casesMap,
		httpPort: httpPort,
		grpcPort: grpcPort,
	}

	mux := http.NewServeMux()
	mux.Handle("/", waf)
	mux.Handle("/graphql", waf)

	waf.httpServer = &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", httpPort),
		Handler: mux,
	}

	grpcSrv := &grpcServer{
		errChan:  errChan,
		casesMap: casesMap,
	}

	waf.grpcServer = grpc.NewServer()
	pb.RegisterServiceFooBarServer(waf.grpcServer, grpcSrv)

	return waf
}

func (waf *WAF) Run() {
	go func() {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", waf.httpPort), time.Second)
		if err == nil {
			if conn != nil {
				conn.Close()
			}
			waf.errChan <- fmt.Errorf("port %d is already in use", waf.httpPort)
		}

		err = waf.httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			waf.errChan <- fmt.Errorf("HTTP listen and serve error: %v", err)
		}
	}()

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", waf.grpcPort))
		if err != nil {
			waf.errChan <- fmt.Errorf("failed to listen for grpc connections: %v", err)
		}
		if err = waf.grpcServer.Serve(lis); err != nil {
			waf.errChan <- fmt.Errorf("failed to serve grpc connections: %v", err)
		}
	}()
}

func (waf *WAF) Shutdown() error {
	err := waf.httpServer.Shutdown(context.Background())
	waf.grpcServer.GracefulStop()
	return err
}

func (waf *WAF) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clientInfo := r.Header.Get("client")
	if clientInfo == "" && !strings.Contains(strings.ToLower(r.URL.String()), "graphql") {
		waf.errChan <- errors.New("couldn't get client info header value")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	caseHash := r.Header.Get(clients.GTWDebugHeader)
	if caseHash == "" {
		if clientInfo == "chrome" {
			w.WriteHeader(http.StatusOK)
			return
		}

		waf.errChan <- errors.New("couldn't get X-GoTestWAF-Test header value")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	payloadInfo, ok := waf.casesMap.CheckTestCaseAvailability(caseHash)
	if !ok {
		waf.errChan <- fmt.Errorf("received unknown case hash: %s", caseHash)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	payloadInfoValues := strings.Split(payloadInfo, ",")

	var err error
	var set string
	var name string
	var placeholder string
	var placeholderValue string
	var encoder string
	var value string

	testCaseParameters := make(map[string]string)

	for _, value = range payloadInfoValues {
		kv := strings.Split(value, "=")

		if len(kv) < 2 {
			waf.errChan <- errors.New("couldn't parse header value")
			w.WriteHeader(http.StatusBadRequest)
			return
		} else {
			testCaseParameters[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}

	if set, ok = testCaseParameters["set"]; !ok {
		waf.errChan <- errors.New("couldn't get `set` parameter of test case")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if name, ok = testCaseParameters["name"]; !ok {
		waf.errChan <- errors.New("couldn't get `name` parameter of test case")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if placeholder, ok = testCaseParameters["placeholder"]; !ok {
		waf.errChan <- errors.New("couldn't get `placeholder` parameter of test case")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if encoder, ok = testCaseParameters["encoder"]; !ok {
		waf.errChan <- errors.New("couldn't get `encoder` parameter of test case")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch placeholder {
	case "GraphQL":
		err = nil
		placeholderValue = config.GraphQLConfigs[set].GetPayloadFunc(r)
	case "Header":
		placeholderValue, err = getPayloadFromHeader(r)
	case "HTMLForm":
		placeholderValue, err = getPayloadFromHTMLForm(r)
	case "HTMLMultipartForm":
		placeholderValue, err = getPayloadFromHTMLMultipartForm(r)
	case "JSONBody":
		placeholderValue, err = getPayloadFromJSONBody(r)
	case "JSONRequest":
		placeholderValue, err = getPayloadFromJSONRequest(r)
	case "RawRequest":
		err = nil
		placeholderValue = config.RawRequestConfigs[set].GetPayloadFunc(r)
	case "RequestBody":
		placeholderValue, err = getPayloadFromRequestBody(r)
	case "SOAPBody":
		placeholderValue, err = getPayloadFromSOAPBody(r)
	case "URLParam":
		placeholderValue, err = getPayloadFromURLParam(r)
	case "URLPath":
		placeholderValue, err = getPayloadFromURLPath(r)
	case "UserAgent":
		placeholderValue, err = getPayloadFromUAHeader(r)
	case "XMLBody":
		placeholderValue, err = getPayloadFromXMLBody(r)
	default:
		waf.errChan <- fmt.Errorf("unknown placeholder: %s", placeholder)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err != nil {
		waf.errChan <- fmt.Errorf("couldn't get encoded payload value: %v, payload info: %s", err, payloadInfo)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch encoder {
	case "Base64":
		value, err = decodeBase64(placeholderValue)
	case "Base64Flat":
		value, err = decodeBase64(placeholderValue)
	case "JSUnicode":
		value, err = decodeJSUnicode(placeholderValue)
	case "Plain":
		value, err = decodePlain(placeholderValue)
	case "URL":
		value, err = decodeURL(placeholderValue)
	case "XMLEntity":
		value, err = decodeXMLEntity(placeholderValue)
	default:
		waf.errChan <- fmt.Errorf("unknown encoder: %s", encoder)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err != nil {
		waf.errChan <- fmt.Errorf("couldn't decode payload: %v, payload info: %s", err, payloadInfo)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if matched, _ := regexp.MatchString("bypassed", value); matched {
		w.WriteHeader(http.StatusOK)
	} else if matched, _ = regexp.MatchString("blocked", value); matched {
		w.WriteHeader(http.StatusForbidden)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}

	hash := sha256.New()
	hash.Write([]byte(set))
	hash.Write([]byte(name))
	hash.Write([]byte(placeholder))
	hash.Write([]byte(encoder))
	hash.Write([]byte(value))
	restoredCaseHash := hex.EncodeToString(hash.Sum(nil))

	if caseHash != restoredCaseHash {
		waf.errChan <- fmt.Errorf("case hash mismatched: %s != %s", caseHash, restoredCaseHash)
		return
	}
}
