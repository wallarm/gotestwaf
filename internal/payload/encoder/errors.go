package encoder

import "fmt"

var _ error = (*UnknownEncoderError)(nil)

type UnknownEncoderError struct {
	name string
}

func (e *UnknownEncoderError) Error() string {
	return fmt.Sprintf("unknown encoder: %s", e.name)
}
