package clients

import (
	"context"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/payload"
	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

var _ HTTPClient = (*ChromeHTTPClient)(nil)

type ChromeHTTPClient struct {
}

func NewChromeHTTPClient() (*ChromeHTTPClient, error) {
	c := &ChromeHTTPClient{}

	return c, nil
}

func (c *ChromeHTTPClient) SendPayload(
	ctx context.Context,
	targetURL string,
	payloadInfo *payload.PayloadInfo,
) (types.Response, error) {
	request, err := payloadInfo.GetRequest(targetURL, types.ChromeHTTPClient)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't prepare request")
	}

	r, ok := request.(*types.ChromeDPTasks)
	if !ok {
		return nil, errors.Errorf("bad request type: %T, expected %T", request, &types.ChromeDPTasks{})
	}

	_ = r

	return nil, nil
}

func (c *ChromeHTTPClient) SendRequest(
	ctx context.Context,
	req types.Request,
) (types.Response, error) {
	r, ok := req.(*types.ChromeDPTasks)
	if !ok {
		return nil, errors.Errorf("bad request type: %T, expected %T", req, &types.ChromeDPTasks{})
	}

	_ = r

	return nil, nil
}
