package placeholder

import "fmt"

var _ error = (*UnknownPlaceholderError)(nil)
var _ error = (*BadPlaceholderConfigError)(nil)

type UnknownPlaceholderError struct {
	name string
}

func (e *UnknownPlaceholderError) Error() string {
	return fmt.Sprintf("unknown placeholder: %s", e.name)
}

type BadPlaceholderConfigError struct {
	name string
	err  error
}

func (e *BadPlaceholderConfigError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("bad config for %s placeholder: %s", e.name, e.err.Error())
	}

	return fmt.Sprintf("bad config for %s placeholder", e.name)
}

func (e *BadPlaceholderConfigError) Unwrap() error {
	return e.err
}
