package encoder

import (
	"encoding/base64"
	"fmt"
	"strings"
)

type Base64Encoder struct {
	name string
	mode uint8
}

const (
	ENC_NORMAL = 1
	ENC_FLAT   = 2
)

var DefaultBase64Encoder = Base64Encoder{name: "Base64", mode: ENC_NORMAL}
var DefaultBase64FlatEncoder = Base64Encoder{name: "Base64Flat", mode: ENC_FLAT}

func (enc Base64Encoder) GetName() *string {
	return &enc.name
}

func (enc Base64Encoder) Encode(data string) (string, error) {
	switch enc.mode {
	case ENC_NORMAL:
		res := base64.StdEncoding.EncodeToString([]byte(data))
		return res, nil
	case ENC_FLAT:
		res := strings.ReplaceAll(base64.StdEncoding.EncodeToString([]byte(data)), "=", "")
		return res, nil
	}
	return "", fmt.Errorf("undefined encoding method")
}
