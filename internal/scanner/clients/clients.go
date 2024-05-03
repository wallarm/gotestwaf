package clients

import (
	"context"

	"github.com/wallarm/gotestwaf/internal/payload"

	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

// HTTPClient is an interface that defines methods for sending HTTP requests and payloads.
type HTTPClient interface {
	// SendPayload sends a payload to the specified target URL.
	SendPayload(ctx context.Context, targetURL string, payloadInfo *payload.PayloadInfo) (types.Response, error)

	// SendRequest sends a prepared custom request to the specified target URL.
	SendRequest(ctx context.Context, req types.Request) (types.Response, error)
}
