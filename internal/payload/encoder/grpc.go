package encoder

type GRPCEncoder struct {
	name string
}

var DefaultGRPCEncoder = GRPCEncoder{name: "gRPC"}

func (enc GRPCEncoder) GetName() string {
	return enc.name
}

func (enc GRPCEncoder) Encode(data string) (string, error) {
	return data, nil
}
