package encoder

var _ Encoder = (*PlainEncoder)(nil)

var DefaultPlainEncoder = &PlainEncoder{name: "Plain"}

type PlainEncoder struct {
	name string
}

func (enc *PlainEncoder) GetName() string {
	return enc.name
}

func (enc *PlainEncoder) Encode(data string) (string, error) {
	return data, nil
}
