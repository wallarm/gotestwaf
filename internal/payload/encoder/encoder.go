package encoder

type Encoder interface {
	GetName() string
	Encode(data string) (string, error)
}

var Encoders map[string]Encoder

func init() {
	Encoders = make(map[string]Encoder)
	Encoders[DefaultBase64Encoder.GetName()] = DefaultBase64Encoder
	Encoders[DefaultBase64FlatEncoder.GetName()] = DefaultBase64FlatEncoder
	Encoders[DefaultJSUnicodeEncoder.GetName()] = DefaultJSUnicodeEncoder
	Encoders[DefaultURLEncoder.GetName()] = DefaultURLEncoder
	Encoders[DefaultPlainEncoder.GetName()] = DefaultPlainEncoder
	Encoders[DefaultXMLEntityEncoder.GetName()] = DefaultXMLEntityEncoder
}

func Apply(encoderName, data string) (string, error) {
	ret, err := Encoders[encoderName].Encode(data)
	if err != nil {
		return "", err
	}
	return ret, nil
}
