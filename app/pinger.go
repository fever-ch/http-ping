package app

import (
	"fmt"
	"time"
)

// Answer is the out of a measurement done as an HTTP ping
type Answer struct {
	Duration     time.Duration
	StatusCode   int
	Bytes        int64
	InBytes      int64
	OutBytes     int64
	SocketReused bool
}

func (a *Answer) String() string {
	return fmt.Sprintf("code=%d size=%d conn-reused=%t time=%.3f ms", a.StatusCode, a.Bytes, a.SocketReused, float64(a.Duration.Nanoseconds())/1e6)
}
