package placeholder

import "net/http"

type GRPC struct {
	name string
}

var DefaultGRPC = GRPC{name: "gRPC"}

var _ Placeholder = (*GRPC)(nil)

func (enc GRPC) newConfig(_ map[any]any) (any, error) {
	return nil, nil
}

func (enc GRPC) GetName() string {
	return enc.name
}

func (enc GRPC) CreateRequest(string, string, any) (*http.Request, error) {
	return nil, nil
}
