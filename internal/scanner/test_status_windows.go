package scanner

import (
	"context"
	"sync/atomic"
	"unsafe"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
)

type k32Event struct {
	keyDown         int32
	repeatCount     uint16
	virtualKeyCode  uint16
	virtualScanCode uint16
	unicodeChar     uint16
	controlKeyState uint32
}

var (
	kernel32              = windows.NewLazyDLL("kernel32.dll")
	k32_ReadConsoleInputW = kernel32.NewProc("ReadConsoleInputW")
)

func keyboardListener(ctx context.Context, c chan struct{}) error {
	hInterrupt, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return err
	}

	hConsoleIn, err := windows.Open("CONIN$", windows.O_RDWR, 0)
	if err != nil {
		windows.Close(hInterrupt)
		return err
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, err := windows.WaitForSingleObject(hConsoleIn, uint32(windows.INFINITE))
				if err != nil {
					continue
				}

				var input [20]uint16
				var numberOfEventsRead uint32
				r0, _, err := k32_ReadConsoleInputW.Call(uintptr(hConsoleIn), uintptr(unsafe.Pointer(&input[0])), 1, uintptr(unsafe.Pointer(&numberOfEventsRead)))
				if int(r0) == 0 {
					continue
				}
				if input[0] == 0x1 {
					kEvent := (*k32Event)(unsafe.Pointer(&input[2]))
					keyCode := kEvent.virtualKeyCode
					ctrlPressed := kEvent.controlKeyState&(0x08|0x04) != 0

					// check if Ctrl + B pressed
					if ctrlPressed && keyCode == 0x42 {
						c <- struct{}{}
					}
				}
			}
		}
	}()
	return nil
}

// On windows, it listens for the Ctrl + B keypress
func (s *Scanner) listeningForPrintTestStatus(ctx context.Context, requestsCounter *uint64) (func(), error) {
	userSignal := make(chan struct{}, 1)
	if err := keyboardListener(ctx, userSignal); err != nil {
		return nil, err
	}
	s.logger.Info("Press Ctrl + B view testing status")
	go func() {
		for {
			select {
			case <-userSignal:
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
		close(userSignal)
	}, nil
}
