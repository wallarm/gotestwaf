package clients

import (
	"context"

	"github.com/wallarm/gotestwaf/internal/payload"
	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

const GTWDebugHeader = "X-GoTestWAF-Test"

// HTTPClient is an interface that defines methods for sending HTTP requests and payloads.
type HTTPClient interface {
	// SendPayload sends a payload to the specified target URL.
	SendPayload(ctx context.Context, targetURL string, payloadInfo *payload.PayloadInfo) (types.Response, error)

	// SendRequest sends a prepared custom request to the specified target URL.
	SendRequest(ctx context.Context, req types.Request) (types.Response, error)
}

// GraphQLClient is an interface that defines methods for sending
// GraphQL payloads by HTTP protocol.
type GraphQLClient interface {
	// CheckAvailability checks availability of endpoint which is able to
	// process GraphQL protocol messages.
	CheckAvailability(ctx context.Context) (bool, error)

	// IsAvailable returns status of endpoint availability.
	IsAvailable() bool

	// SendPayload sends a payload to the specified target URL.
	SendPayload(ctx context.Context, payloadInfo *payload.PayloadInfo) (types.Response, error)
}

// GRPCClient is an interface that defines methods for sending
// payloads by gRPC protocol.
type GRPCClient interface {
	// CheckAvailability checks availability of endpoint which is able to
	// process gRPC protocol messages.
	CheckAvailability(ctx context.Context) (bool, error)

	// IsAvailable returns status of endpoint availability.
	IsAvailable() bool

	// SendPayload sends a payload to the specified target URL.
	SendPayload(ctx context.Context, payloadInfo *payload.PayloadInfo) (types.Response, error)

	// Close closes underlying connection.
	Close() error
}
