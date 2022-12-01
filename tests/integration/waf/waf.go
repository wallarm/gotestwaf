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

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"

	pb "github.com/wallarm/gotestwaf/internal/payload/placeholder/grpc"
	"github.com/wallarm/gotestwaf/internal/scanner"
	"github.com/wallarm/gotestwaf/tests/integration/config"
)

var _ http.Handler = &WAF{}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WAF struct {
	errChan    chan<- error
	casesMap   *config.TestCasesMap
	httpServer *http.Server
	grpcServer *grpc.Server
}

func New(errChan chan<- error, casesMap *config.TestCasesMap) *WAF {
	waf := &WAF{
		errChan:  errChan,
		casesMap: casesMap,
	}

	mux := http.NewServeMux()
	mux.Handle("/", waf)

	waf.httpServer = &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", config.HTTPPort),
		Handler: mux,
	}

	grpcServer := &grpcServer{
		errChan:  errChan,
		casesMap: casesMap,
	}

	waf.grpcServer = grpc.NewServer()
	pb.RegisterServiceFooBarServer(waf.grpcServer, grpcServer)

	return waf
}

func (waf *WAF) Run() {
	go func() {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", config.HTTPPort), time.Second)
		if err == nil {
			if conn != nil {
				conn.Close()
			}
			waf.errChan <- fmt.Errorf("port %d is already in use", config.HTTPPort)
		}

		err = waf.httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			waf.errChan <- fmt.Errorf("HTTP listen and serve error: %v", err)
		}
	}()

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", config.GRPCPort))
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
	upgrade := false
	for _, header := range r.Header["Upgrade"] {
		if header == "websocket" {
			upgrade = true
			break
		}
	}

	if upgrade == false {
		waf.httpRequestHandler(w, r)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		waf.errChan <- fmt.Errorf("couldn't upgrage http connection to ws: %v", err)
		return
	}

	waf.websocketRequestHandler(ws)
}

func (waf *WAF) httpRequestHandler(w http.ResponseWriter, r *http.Request) {
	caseHash := r.Header.Get(scanner.GTWDebugHeader)
	if caseHash == "" {
		waf.errChan <- errors.New("couldn't get X-GoTestWAF-Test header value")
	}

	payloadInfo, ok := waf.casesMap.CheckTestCaseAvailability(caseHash)
	if !ok {
		waf.errChan <- fmt.Errorf("received unknown case hash: %s", caseHash)
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
		} else {
			testCaseParameters[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}

	if set, ok = testCaseParameters["set"]; !ok {
		waf.errChan <- errors.New("couldn't get `set` parameter of test case")
	}

	if name, ok = testCaseParameters["name"]; !ok {
		waf.errChan <- errors.New("couldn't get `name` parameter of test case")
	}

	if placeholder, ok = testCaseParameters["placeholder"]; !ok {
		waf.errChan <- errors.New("couldn't get `placeholder` parameter of test case")
	}

	if encoder, ok = testCaseParameters["encoder"]; !ok {
		waf.errChan <- errors.New("couldn't get `encoder` parameter of test case")
	}

	switch placeholder {
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
	case "RequestBody":
		placeholderValue, err = getPayloadFromRequestBody(r)
	case "SOAPBody":
		placeholderValue, err = getPayloadFromSOAPBody(r)
	case "URLParam":
		placeholderValue, err = getPayloadFromURLParam(r)
	case "URLPath":
		placeholderValue, err = getPayloadFromURLPath(r)
	case "XMLBody":
		placeholderValue, err = getPayloadFromXMLBody(r)
	case "NonCrudUrlParam":
		placeholderValue, err = getPayloadFromURLParam(r)
	case "NonCrudUrlPath":
		placeholderValue, err = getPayloadFromURLPath(r)
	case "NonCRUDHeader":
		placeholderValue, err = getPayloadFromHeader(r)
	case "NonCRUDRequestBody":
		placeholderValue, err = getPayloadFromRequestBody(r)
	default:
		waf.errChan <- fmt.Errorf("unknown placeholder: %s", placeholder)
	}

	if err != nil {
		waf.errChan <- fmt.Errorf("couldn't get encoded payload value: %v", err)
	}

	switch encoder {
	case "Base64":
		value, err = decodeBase64(placeholderValue)
	case "Base64Flat":
		value, err = decodeBase64(placeholderValue)
	case "JSUnicode":
		value, err = decodeJSUnicode(placeholderValue)
	case "URL":
		value, err = decodeURL(placeholderValue)
	case "Plain":
		value, err = decodePlain(placeholderValue)
	case "XMLEntity":
		value, err = decodeXMLEntity(placeholderValue)
	default:
		waf.errChan <- fmt.Errorf("unknown encoder: %s", encoder)
	}

	if err != nil {
		waf.errChan <- fmt.Errorf("couldn't decode payload: %v", err)
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
	}
}

func (waf *WAF) websocketRequestHandler(conn *websocket.Conn) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
				return
			}
			waf.errChan <- fmt.Errorf("couldn't read message from websocket: %v", err)
			return
		}

		if matched, _ := regexp.MatchString("alert", string(msg)); matched {
			conn.Close()
			return
		}

		err = conn.WriteMessage(websocket.TextMessage, []byte("OK"))
		if err != nil {
			waf.errChan <- fmt.Errorf("couldn't send message to websocket: %v", err)
			return
		}
	}
}
