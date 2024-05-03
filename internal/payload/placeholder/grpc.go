package placeholder

import (
	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

var _ Placeholder = (*GRPC)(nil)

var DefaultGRPC = &GRPC{name: "gRPC"}

type GRPC struct {
	name string
}

func (enc *GRPC) NewPlaceholderConfig(map[any]any) (PlaceholderConfig, error) {
	return nil, nil
}

func (enc *GRPC) GetName() string {
	return enc.name
}

func (enc *GRPC) CreateRequest(string, string, PlaceholderConfig, types.HTTPClientType) (types.Request, error) {
	return nil, nil
}
