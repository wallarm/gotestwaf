package scanner

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	"github.com/wallarm/gotestwaf/internal/config"
	"github.com/wallarm/gotestwaf/internal/payload/encoder"
	grpcSrv "github.com/wallarm/gotestwaf/internal/payload/encoder/grpc"
)

const (
	grpcContentType            = "application/grpc"
	grpcUserAgent              = "grpc-go/1.42.0"
	grpcServerDetectionTimeout = 3 * time.Second
)

type GRPCConn struct {
	host           string
	transportCreds credentials.TransportCredentials
	tlsConf        *tls.Config

	conn *grpc.ClientConn

	isAvailable bool
}

func NewGRPCConn(cfg *config.Config) (*GRPCConn, error) {
	isTLS, host, err := tlsAndHostFromUrl(cfg.URL)
	if err != nil {
		return nil, err
	}

	g := &GRPCConn{host: host}

	if isTLS {
		g.tlsConf = &tls.Config{InsecureSkipVerify: !cfg.TLSVerify}
		g.transportCreds = credentials.NewTLS(g.tlsConf)
	}

	return g, nil
}

func (g *GRPCConn) httpTest() (bool, error) {
	var http2transport *http2.Transport
	var scheme string

	if g.tlsConf == nil {
		http2transport = &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(netw, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(netw, addr)
			},
			DisableCompression: true,
		}

		scheme = "http"
	} else {
		http2transport = &http2.Transport{
			TLSClientConfig:    g.tlsConf,
			DisableCompression: true,
		}

		scheme = "https"
	}

	http2client := &http.Client{Transport: http2transport}

	req := &http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: scheme,
			Host:   g.host,
			Path:   "/",
		},
		Header: http.Header{},
		Body:   io.NopCloser(bytes.NewReader(nil)),
	}
	req.Header.Set("Content-Type", grpcContentType)
	req.Header.Set("User-Agent", grpcUserAgent)

	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), grpcServerDetectionTimeout)
	defer cancel()

	// Sends the request
	httpResp, err := http2client.Do(req.WithContext(ctxWithTimeout))
	if err != nil {
		return false, err
	}
	httpResp.Body.Close()

	if strings.Contains(httpResp.Header.Get("Content-Type"), "application/grpc") {
		return true, nil
	}

	return false, nil
}

func (g *GRPCConn) healthCheckTest() (bool, error) {
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), grpcServerDetectionTimeout)
	defer cancel()

	var (
		conn *grpc.ClientConn
		err  error
	)

	if g.transportCreds == nil {
		conn, err = grpc.DialContext(ctxWithTimeout, g.host, grpc.WithInsecure(), grpc.WithBlock())
	} else {
		conn, err = grpc.DialContext(ctxWithTimeout, g.host, grpc.WithTransportCredentials(g.transportCreds), grpc.WithBlock())
	}

	if err != nil {
		return false, errors.Wrap(err, "sending gRPC request")
	}
	defer conn.Close()

	_, err = healthpb.NewHealthClient(conn).Check(ctxWithTimeout,
		&healthpb.HealthCheckRequest{Service: "ServiceFooBar"})
	if err != nil {
		_, ok := status.FromError(err)
		if ok {
			return true, nil
		}
	}

	return false, nil
}

func (g *GRPCConn) CheckAvailability() (bool, error) {
	ok, err := g.httpTest()
	if err != nil {
		return false, errors.Wrap(err, "checking gRPC availability via HTTP test")
	}

	if ok {
		ok, err = g.healthCheckTest()
		if err != nil {
			return false, errors.Wrap(err, "checking gRPC availability via gRPC health check")
		}
	}

	g.isAvailable = ok

	return ok, nil
}

func (g *GRPCConn) Send(ctx context.Context, encoderName, payload string) (body []byte, statusCode int, err error) {
	if !g.isAvailable {
		return nil, 0, nil
	}

	encodedPayload, err := encoder.Apply(encoderName, payload)
	if err != nil {
		return nil, 0, errors.Wrap(err, "encoding payload")
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, grpcServerDetectionTimeout)
	defer cancel()

	var conn *grpc.ClientConn

	// Set up a connection to the server.
	if g.conn == nil {
		switch g.transportCreds {
		case nil:
			conn, err = grpc.DialContext(ctxWithTimeout, g.host, grpc.WithInsecure(), grpc.WithBlock())
		default:
			conn, err = grpc.DialContext(ctxWithTimeout, g.host, grpc.WithTransportCredentials(g.transportCreds), grpc.WithBlock())
		}
		if err != nil {
			return nil, 0, errors.Wrap(err, "sending gRPC request")
		}

		g.conn = conn
	}

	client := grpcSrv.NewServiceFooBarClient(conn)

	resp, err := client.Foo(ctx, &grpcSrv.Request{Value: encodedPayload})
	if err != nil {
		st := status.Convert(err)

		// gRPC status code converting to HTTP status code
		switch st.Code() {
		case codes.OK:
			statusCode = 200
		case codes.Canceled:
			statusCode = 499
		case codes.Unknown:
			statusCode = 500
		case codes.InvalidArgument:
			statusCode = 400
		case codes.DeadlineExceeded:
			statusCode = 504
		case codes.NotFound:
			statusCode = 404
		case codes.AlreadyExists:
			statusCode = 409
		case codes.PermissionDenied:
			statusCode = 403
		case codes.ResourceExhausted:
			statusCode = 429
		case codes.FailedPrecondition:
			statusCode = 400
		case codes.Aborted:
			statusCode = 409
		case codes.OutOfRange:
			statusCode = 400
		case codes.Unimplemented:
			statusCode = 501
		case codes.Internal:
			statusCode = 500
		case codes.Unavailable:
			statusCode = 503
		case codes.DataLoss:
			statusCode = 500
		case codes.Unauthenticated:
			statusCode = 401
		default:
			statusCode = 500
		}

		return nil, statusCode, nil
	}

	return []byte(resp.GetValue()), 200, nil
}

func (g *GRPCConn) IsAvailable() bool {
	return g.isAvailable
}

func (g *GRPCConn) Close() error {
	if g.conn == nil {
		return nil
	}

	return g.conn.Close()
}

// returns isTLS, URL host:port, error
func tlsAndHostFromUrl(wafURL string) (bool, string, error) {
	isTLS := false

	urlParse, err := url.Parse(wafURL)
	if err != nil {
		return isTLS, "", err
	}

	host := urlParse.Host
	if urlParse.Port() == "" {
		switch urlParse.Scheme {
		case "http":
			host += ":80"
		case "https":
			host += ":443"
		}
	}

	if urlParse.Scheme == "https" {
		isTLS = true
	}

	return isTLS, host, nil
}
