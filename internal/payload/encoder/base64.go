package encoder

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const (
	Base64EncoderNormalMode = iota
	Base64EncoderFlatMode
)

var _ Encoder = (*Base64Encoder)(nil)

var DefaultBase64Encoder = &Base64Encoder{name: "Base64", mode: Base64EncoderNormalMode}
var DefaultBase64FlatEncoder = &Base64Encoder{name: "Base64Flat", mode: Base64EncoderFlatMode}

type Base64Encoder struct {
	name string
	mode uint8
}

func (enc *Base64Encoder) GetName() string {
	return enc.name
}

func (enc *Base64Encoder) Encode(data string) (string, error) {
	switch enc.mode {
	case Base64EncoderNormalMode:
		res := base64.StdEncoding.EncodeToString([]byte(data))
		return res, nil
	case Base64EncoderFlatMode:
		res := strings.ReplaceAll(base64.StdEncoding.EncodeToString([]byte(data)), "=", "")
		return res, nil
	}
	return "", fmt.Errorf("undefined encoding method")
}
