package encoder

type Encoder interface {
	GetName() string
	Encode(data string) (string, error)
}

var Encoders map[string]Encoder

var encoders = []Encoder{
	DefaultBase64Encoder,
	DefaultBase64FlatEncoder,
	DefaultJSUnicodeEncoder,
	DefaultPlainEncoder,
	DefaultURLEncoder,
	DefaultXMLEntityEncoder,
}

func init() {
	Encoders = make(map[string]Encoder)
	for _, encoder := range encoders {
		Encoders[encoder.GetName()] = encoder
	}
}

func Apply(encoderName, data string) (string, error) {
	en, ok := Encoders[encoderName]
	if !ok {
		return "", &UnknownEncoderError{name: encoderName}
	}

	ret, err := en.Encode(data)
	if err != nil {
		return "", err
	}

	return ret, nil
}
