package scanner

import (
	"context"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/sirupsen/logrus"
)

func (s *Scanner) testStatusSignalHandler(ctx context.Context, requestsCounter *uint64) (func(), error) {
	userSignal := make(chan os.Signal, 1)
	signal.Notify(userSignal, syscall.SIGUSR1)

	go func() {
		for {
			select {
			case _, ok := <-userSignal:
				if !ok {
					return
				}

				s.logger.
					WithFields(logrus.Fields{
						"sent":  atomic.LoadUint64(requestsCounter),
						"total": s.db.NumberOfTests,
					}).Info("Testing status")

			case <-ctx.Done():
				return
			}
		}
	}()

	return func() {
		signal.Stop(userSignal)
		close(userSignal)
	}, nil
}
