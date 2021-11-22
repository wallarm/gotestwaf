package waf

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"

	pb "github.com/wallarm/gotestwaf/internal/payload/encoder/grpc"
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
		Addr:    config.HTTPAddress,
		Handler: mux,
	}

	waf.grpcServer = grpc.NewServer()
	pb.RegisterServiceFooBarServer(waf.grpcServer, &grpcServer{})

	return waf
}

func (waf *WAF) Run() {
	go func() {
		conn, err := net.DialTimeout("tcp", config.HTTPAddress, time.Second)
		if err == nil {
			if conn != nil {
				conn.Close()
			}
			waf.errChan <- fmt.Errorf("port %s is already in use", config.HTTPPort)
		}

		err = waf.httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			waf.errChan <- fmt.Errorf("HTTP listen and serve error: %v", err)
		}
	}()

	go func() {
		lis, err := net.Listen("tcp", config.GRPCAddress)
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
	headerValues := strings.Split(r.Header.Get("X-GoTestWAF-Test"), ",")
	if headerValues == nil {
		waf.errChan <- errors.New("couldn't get X-GoTestWAF-Test header value")
	}

	var err error
	var ok bool
	var set string
	var name string
	var placeholder string
	var placeholderValue string
	var encoder string
	var value string

	testCaseParameters := make(map[string]string)

	for _, value = range headerValues {
		kv := strings.Split(value, "=")

		if len(kv) != 2 {
			waf.errChan <- errors.New("couldn't parse X-GoTestWAF-Test header value")
		}

		testCaseParameters[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
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
	case "GRPC":
		value, err = decodeGRPC(placeholderValue)
	default:
		waf.errChan <- fmt.Errorf("unknown encoder: %s", encoder)
	}

	if err != nil {
		waf.errChan <- fmt.Errorf("couldn't decode payload: %v", err)
	}

	if matched, _ := regexp.MatchString("bypassed", value); matched {
		w.WriteHeader(http.StatusNonAuthoritativeInfo)
	} else if matched, _ = regexp.MatchString("blocked", value); matched {
		w.WriteHeader(http.StatusForbidden)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}

	testCase := fmt.Sprintf("%s-%s-%s-%s-%s", set, name, value, placeholder, encoder)
	if !waf.casesMap.CheckTestCaseAvailability(testCase) {
		waf.errChan <- fmt.Errorf("received unknown payload: %s", testCase)
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
