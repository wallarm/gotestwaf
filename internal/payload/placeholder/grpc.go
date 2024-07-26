package placeholder

import (
	"errors"

	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

var _ Placeholder = (*GRPC)(nil)

var DefaultGRPC = &GRPC{name: "gRPC"}

type GRPC struct {
	name string
}

func (p *GRPC) NewPlaceholderConfig(map[any]any) (PlaceholderConfig, error) {
	return nil, nil
}

func (p *GRPC) GetName() string {
	return p.name
}

func (p *GRPC) CreateRequest(string, string, PlaceholderConfig, types.HTTPClientType) (types.Request, error) {
	return nil, errors.New("not implemented")
}
