package scanner

import (
	"context"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/wallarm/gotestwaf/internal/data/config"
	"github.com/wallarm/gotestwaf/internal/payload/encoder"
	grpcSrv "github.com/wallarm/gotestwaf/internal/payload/encoder/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCData struct {
	host string
	available bool
}

func NewGRPCData(cfg *config.Config) (*GRPCData, error) {

	host, err := hostFromUrl(cfg.URL)
	if err != nil {
		return nil ,err
	}

	return &GRPCData{
		host:        host,
		available: false,
	}, nil
}

func (g *GRPCData) CheckAvailability() (bool, error) {
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set up a connection to the server.
	conn, err := grpc.DialContext(ctxWithTimeout, g.host, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDisableRetry())
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

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Set up a connection to the server.
	conn, err := grpc.DialContext(ctxWithTimeout, g.host, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDisableRetry())
	if err != nil {
		return nil, 0, errors.Wrap(err, "sending gRPC request")
	}
	defer conn.Close()

	client := grpcSrv.NewServiceFooBarClient(conn)

	resp, err := client.Foo(ctx, &grpcSrv.Request{Value: encodedPayload})
	if err != nil {
		st := status.Convert(err)

		// gRPC status code converting to HTTP status code
		switch st.Code() {
		case codes.OK:
			statusCode=200
		case codes.Canceled:
			statusCode=499
		case codes.Unknown:
			statusCode=500
		case codes.InvalidArgument:
			statusCode=400
		case codes.DeadlineExceeded:
			statusCode=504
		case codes.NotFound:
			statusCode=404
		case codes.AlreadyExists:
			statusCode=409
		case codes.PermissionDenied:
			statusCode=403
		case codes.ResourceExhausted:
			statusCode=429
		case codes.FailedPrecondition:
			statusCode=400
		case codes.Aborted:
			statusCode=409
		case codes.OutOfRange:
			statusCode=400
		case codes.Unimplemented:
			statusCode=501
		case codes.Internal:
			statusCode=500
		case codes.Unavailable:
			statusCode=503
		case codes.DataLoss:
			statusCode=500
		case codes.Unauthenticated:
			statusCode=401
		default:
			statusCode=500
		}

		return nil, statusCode, nil
	}

	return []byte(resp.GetValue()), 200, nil
}

func (g *GRPCData) UpdateGRPCData(available bool) {
	g.available = available
}

func hostFromUrl(wafURL string) (string, error) {
	urlParse, err := url.Parse(wafURL)
	if err != nil {
		return "", err
	}

	host := urlParse.Hostname()
	if urlParse.Port() == "" {
		switch urlParse.Scheme {
		case "http":
			host += ":80"
		case "https":
			host += ":443"
		}
	}
	return host, nil
}