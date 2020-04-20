package encoder

/* Better to use byte types, they're faster, but here I'll use strings */
type Encoder interface {
	GetName() *string
	Encode(data string) (string, error)
}

var Encoders map[string]Encoder

func InitEncoders() {
	Encoders = make(map[string]Encoder)
	Encoders[*DefaultBase64Encoder.GetName()] = DefaultBase64Encoder
	Encoders[*DefaultBase64FlatEncoder.GetName()] = DefaultBase64FlatEncoder
	Encoders[*DefaultJSUnicodeEncoder.GetName()] = DefaultJSUnicodeEncoder
	Encoders[*DefaultUrlEncoder.GetName()] = DefaultUrlEncoder
	Encoders[*DefaultPlainEncoder.GetName()] = DefaultPlainEncoder
	Encoders[*DefaultXmlEntityEncoder.GetName()] = DefaultXmlEntityEncoder
}

func Apply(encoderName string, data string) (string, error) {
	return Encoders[encoderName].Encode(data)
}
