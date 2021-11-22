package util

import (
	"os"
	"os/signal"
	"time"
)

// signalHandler is a helper structure which helps to intercept signals
type signalHandler struct {
	triggered bool
}

// NewSignalHandler builds a new signalHandler which will be registered to the signal sig (i.e. os.Interrupt)
func NewSignalHandler(sig os.Signal) *signalHandler {
	c := make(chan os.Signal, 1)

	h := signalHandler{
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
func (sh *signalHandler) Triggered() bool {
	return sh.triggered
}

// Sleep does sleep until duration elapses or a signal is triggered
func (sh *signalHandler) Sleep(duration time.Duration) {
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
