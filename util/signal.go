package util

import (
	"os"
	"os/signal"
	"time"
)

// SignalHandler is a helper structure which helps to intercept signals
type SignalHandler struct {
	triggered bool
}

// NewSignalHandler builds a new SignalHandler which will be registered to the signal sig (i.e. os.Interrupt)
func NewSignalHandler(sig os.Signal) *SignalHandler {
	c := make(chan os.Signal, 1)

	h := SignalHandler{
		triggered: false,
	}

	signal.Notify(c, sig)
	go func() {
		<-c
		h.triggered = true
	}()

	return &h
}

// Triggered returns true if the selected signal has been triggered
func (sh *SignalHandler) Triggered() bool {
	return sh.triggered
}

// Sleep does sleep until duration elapses or a signal is triggered
func (sh *SignalHandler) Sleep(duration time.Duration) {
	deadline := time.Now().Add(duration)
	for !sh.triggered {
		toSleep := time.Until(deadline)
		if toSleep <= 0 {
			break
		} else if toSleep > time.Millisecond*10 {
			toSleep = time.Millisecond * 10
		}
		time.Sleep(toSleep)
	}
}
