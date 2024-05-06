package clients

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
	"github.com/wallarm/gotestwaf/internal/helpers"
	"github.com/wallarm/gotestwaf/internal/payload"
	grpcPlaceholder "github.com/wallarm/gotestwaf/internal/payload/placeholder/grpc"
	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

const (
	grpcContentType            = "application/grpc"
	grpcUserAgent              = "grpc-go/1.56.0"
	grpcServerDetectionTimeout = 3 * time.Second
)

var _ GRPCClient = (*GrpcClient)(nil)

type GrpcClient struct {
	host           string
	transportCreds credentials.TransportCredentials
	tlsConf        *tls.Config

	conn *grpc.ClientConn

	isAvailable bool
}

func NewGrpcClient(cfg *config.Config) (*GrpcClient, error) {
	g := &GrpcClient{isAvailable: true}

	if cfg.GRPCPort == 0 {
		g.isAvailable = false
		return g, nil
	}

	isTLS, host, err := helpers.HostPortFromUrl(cfg.URL, cfg.GRPCPort)
	if err != nil {
		return nil, err
	}

	g.host = host

	if isTLS {
		g.tlsConf = &tls.Config{InsecureSkipVerify: !cfg.TLSVerify}
		g.transportCreds = credentials.NewTLS(g.tlsConf)
	}

	return g, nil
}

func (g *GrpcClient) httpTest(ctx context.Context) (bool, error) {
	var http2transport *http2.Transport
	var scheme string

	if g.tlsConf == nil {
		http2transport = &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network string, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
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

	ctxWithTimeout, cancel := context.WithTimeout(ctx, grpcServerDetectionTimeout)
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

func (g *GrpcClient) healthCheckTest(ctx context.Context) (bool, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, grpcServerDetectionTimeout)
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

func (g *GrpcClient) CheckAvailability(ctx context.Context) (bool, error) {
	if !g.isAvailable {
		return false, nil
	}

	ok, err := g.httpTest(ctx)
	if err != nil {
		g.isAvailable = false
		return false, errors.Wrap(err, "checking gRPC availability via HTTP test")
	}

	if ok {
		ok, err = g.healthCheckTest(ctx)
		if err != nil {
			g.isAvailable = false
			return false, errors.Wrap(err, "checking gRPC availability via gRPC health check")
		}
	}

	g.isAvailable = ok

	return ok, nil
}

func (g *GrpcClient) IsAvailable() bool {
	return g.isAvailable
}

func (g *GrpcClient) SendPayload(ctx context.Context, payloadInfo *payload.PayloadInfo) (types.Response, error) {
	if !g.isAvailable {
		return nil, nil
	}

	encodedPayload, err := payloadInfo.GetEncodedPayload()
	if err != nil {
		return nil, errors.Wrap(err, "encoding payload")
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, grpcServerDetectionTimeout)
	defer cancel()

	// Set up a connection to the server.
	if g.conn == nil {
		var conn *grpc.ClientConn

		switch g.transportCreds {
		case nil:
			conn, err = grpc.DialContext(ctxWithTimeout, g.host, grpc.WithInsecure(), grpc.WithBlock())
		default:
			conn, err = grpc.DialContext(ctxWithTimeout, g.host, grpc.WithTransportCredentials(g.transportCreds), grpc.WithBlock())
		}
		if err != nil {
			return nil, errors.Wrap(err, "sending gRPC request")
		}

		g.conn = conn
	}

	client := grpcPlaceholder.NewServiceFooBarClient(g.conn)

	response := &types.ResponseMeta{
		StatusCode: 200,
	}

	resp, err := client.Foo(ctx, &grpcPlaceholder.Request{Value: encodedPayload})
	if err != nil {
		st := status.Convert(err)

		// gRPC status code converting to HTTP status code
		switch st.Code() {
		case codes.OK:
			response.StatusCode = 200
		case codes.Canceled:
			response.StatusCode = 499
		case codes.Unknown:
			response.StatusCode = 500
		case codes.InvalidArgument:
			response.StatusCode = 400
		case codes.DeadlineExceeded:
			response.StatusCode = 504
		case codes.NotFound:
			response.StatusCode = 404
		case codes.AlreadyExists:
			response.StatusCode = 409
		case codes.PermissionDenied:
			response.StatusCode = 403
		case codes.ResourceExhausted:
			response.StatusCode = 429
		case codes.FailedPrecondition:
			response.StatusCode = 400
		case codes.Aborted:
			response.StatusCode = 409
		case codes.OutOfRange:
			response.StatusCode = 400
		case codes.Unimplemented:
			response.StatusCode = 501
		case codes.Internal:
			response.StatusCode = 500
		case codes.Unavailable:
			response.StatusCode = 503
		case codes.DataLoss:
			response.StatusCode = 500
		case codes.Unauthenticated:
			response.StatusCode = 401
		default:
			response.StatusCode = 500
		}

		return response, nil
	}

	response.Content = []byte(resp.GetValue())

	return response, nil
}

func (g *GrpcClient) Close() error {
	if g.conn == nil {
		return nil
	}

	return g.conn.Close()
}
