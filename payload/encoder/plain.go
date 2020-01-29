package encoder

type PlainEncoder struct {
	name string
}

var DefaultPlainEncoder = Base64Encoder{name: "Plain"}

func (enc PlainEncoder) GetName() *string {
	return &enc.name
}

func (enc PlainEncoder) Encode(data string) (string, error) {
	return data, nil
}
