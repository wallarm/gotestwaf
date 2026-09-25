package scanner

import (
	"context"
	"io"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/wallarm/gotestwaf/internal/config"
	"github.com/wallarm/gotestwaf/internal/db"
	"github.com/wallarm/gotestwaf/internal/payload"
	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

type fakeProtocolClient struct {
	available bool
	checks    int
}

func (f *fakeProtocolClient) CheckAvailability(context.Context) (bool, error) {
	f.checks++
	return f.available, nil
}
func (f *fakeProtocolClient) IsAvailable() bool { return f.available }
func (f *fakeProtocolClient) SendPayload(context.Context, *payload.PayloadInfo) (types.Response, error) {
	return nil, nil
}
func (f *fakeProtocolClient) Close() error { return nil }

func newTestScanner(cfg *config.Config, grpc, graphql *fakeProtocolClient) *Scanner {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	return &Scanner{logger: logger, cfg: cfg, db: &db.DB{}, grpcConn: grpc, graphqlClient: graphql}
}

func TestGraphQLPreCheckDoesNotOverwriteGRPCAvailability(t *testing.T) {
	s := newTestScanner(&config.Config{}, &fakeProtocolClient{available: true}, &fakeProtocolClient{available: false})

	s.CheckGRPCAvailability(context.Background())
	s.CheckGraphQLAvailability(context.Background())

	if !s.db.IsGrpcAvailable {
		t.Error("gRPC availability was overwritten by the GraphQL pre-check")
	}
	if s.db.IsGraphQLAvailable {
		t.Error("GraphQL should be reported as not available")
	}
}

func TestPreChecksCanBeSkipped(t *testing.T) {
	grpc := &fakeProtocolClient{available: true}
	graphql := &fakeProtocolClient{available: true}
	s := newTestScanner(&config.Config{SkipGRPCCheck: true, SkipGraphQLCheck: true}, grpc, graphql)

	s.CheckGRPCAvailability(context.Background())
	s.CheckGraphQLAvailability(context.Background())

	if grpc.checks != 0 || graphql.checks != 0 {
		t.Errorf("pre-checks ran despite skip flags: grpc=%d graphql=%d", grpc.checks, graphql.checks)
	}
	if !s.db.IsGrpcAvailable || !s.db.IsGraphQLAvailable {
		t.Error("skipped pre-checks must take availability from the clients")
	}
}
