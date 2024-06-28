package payload

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/payload/encoder"
	"github.com/wallarm/gotestwaf/internal/payload/placeholder"
	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

// PayloadInfo holds information about the payload and its configuration.
type PayloadInfo struct {
	Payload           string
	EncoderName       string
	PlaceholderName   string
	PlaceholderConfig placeholder.PlaceholderConfig

	DebugHeaderValue string
}

// GetEncodedPayload encodes the payload using the specified encoder and
// returns the encoded payload.
func (p *PayloadInfo) GetEncodedPayload() (string, error) {
	encodedPayload, err := encoder.Apply(p.EncoderName, p.Payload)
	if err != nil {
		return "", errors.Wrap(err, "couldn't encode payload")
	}

	return encodedPayload, nil
}

// GetRequest generates a request based on the payload information, target URL,
// and client type.
func (p *PayloadInfo) GetRequest(targetURL string, clientType types.HTTPClientType) (types.Request, error) {
	encodedPayload, err := p.GetEncodedPayload()
	if err != nil {
		return nil, err
	}

	request, err := placeholder.Apply(
		targetURL,
		encodedPayload,
		p.PlaceholderName,
		p.PlaceholderConfig,
		clientType,
	)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("couldn't apply placeholder %s", p.PlaceholderName))
	}

	switch r := request.(type) {
	case *types.GoHTTPRequest:
		r.DebugHeaderValue = p.DebugHeaderValue
	case *types.ChromeDPTasks:
		r.DebugHeaderValue = p.DebugHeaderValue
	}

	return request, nil
}
