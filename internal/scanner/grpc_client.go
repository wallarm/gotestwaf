package scanner

import (
	"context"
	"crypto/tls"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/wallarm/gotestwaf/internal/data/config"
	"github.com/wallarm/gotestwaf/internal/payload/encoder"
	grpcSrv "github.com/wallarm/gotestwaf/internal/payload/encoder/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

const GRPCServerDetectionTimeout = 3

type GRPCData struct {
	host      string
	available bool
	tc        credentials.TransportCredentials
}

func NewGRPCData(cfg *config.Config) (*GRPCData, error) {

	var tc credentials.TransportCredentials

	isTLS, host, err := tlsAndHostFromUrl(cfg.URL)
	if err != nil {
		return nil, err
	}

	if isTLS {
		tc = credentials.NewTLS(&tls.Config{InsecureSkipVerify: !cfg.TLSVerify})
	} else {
		tc = nil
	}

	return &GRPCData{
		host:      host,
		available: false,
		tc:        tc,
	}, nil
}

func (g *GRPCData) CheckAvailability() (bool, error) {
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), GRPCServerDetectionTimeout*time.Second)
	defer cancel()

	var (
		conn *grpc.ClientConn
		err  error
	)

	// Set up a connection to the server.
	switch g.tc {
	case nil:
		conn, err = grpc.DialContext(ctxWithTimeout, g.host, grpc.WithInsecure(), grpc.WithBlock())
	default:
		conn, err = grpc.DialContext(ctxWithTimeout, g.host, grpc.WithTransportCredentials(g.tc), grpc.WithBlock())
	}

	if err != nil {
		return false, errors.Wrap(err, "sending gRPC request")
	}
	defer conn.Close()

	return true, nil
}

func (g *GRPCData) SetAvailability(status bool) {
	g.available = status
}

func (g *GRPCData) Send(ctx context.Context, encoderName, payload string) (body []byte, statusCode int, err error) {

	if !g.available {
		return nil, 0, nil
	}

	encodedPayload, err := encoder.Apply(encoderName, payload)
	if err != nil {
		return nil, 0, errors.Wrap(err, "encoding payload")
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, GRPCServerDetectionTimeout*time.Second)
	defer cancel()

	var conn *grpc.ClientConn

	// Set up a connection to the server.
	switch g.tc {
	case nil:
		conn, err = grpc.DialContext(ctxWithTimeout, g.host, grpc.WithInsecure(), grpc.WithBlock())
	default:
		conn, err = grpc.DialContext(ctxWithTimeout, g.host, grpc.WithTransportCredentials(g.tc), grpc.WithBlock())
	}
	if err != nil {
		return nil, 0, errors.Wrap(err, "sending gRPC request")
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

func (g *GRPCData) UpdateGRPCData(available bool) {
	g.available = available
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
