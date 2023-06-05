package report

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var _ error = (*ValidationError)(nil)

type ValidationError struct {
	errors validator.ValidationErrors
}

func (e *ValidationError) Error() string {
	errMsgs := make([]string, len(e.errors))
	for i, fe := range e.errors {
		errMsg := fe.Error()
		if _, ok := customValidators[fe.Tag()]; ok {
			errMsg = fmt.Sprintf("%s, bad value: '%s'", errMsg, fe.Value())
		}

		errMsgs[i] = errMsg
	}

	msg := fmt.Sprintf("found invalid values in the report data: %s", strings.Join(errMsgs, "; "))

	return msg
}
