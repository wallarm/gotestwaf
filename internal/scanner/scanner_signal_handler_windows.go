package scanner

import (
	"context"
)

func (s *Scanner) testStatusSignalHandler(_ context.Context, _ *uint64) (func(), error) {
	return func() {}, nil
}
