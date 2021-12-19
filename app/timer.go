package app

import (
	"github.com/fever-ch/http-ping/stats"
	"math"
	"time"
)

var (
	defaultStartTime = time.UnixMicro(math.MaxInt64)
	defaultStopTime  = time.UnixMicro(math.MinInt64)
)

type timer struct {
	startTime, stopTime time.Time
}

func newTimer() *timer {
	return &timer{
		defaultStartTime,
		defaultStopTime,
	}
}

func (t *timer) start() {
	ts := time.Now()
	if ts.Before(t.startTime) {
		t.startTime = ts
	}
}

func (t *timer) stop() {
	ts := time.Now()
	if ts.After(t.stopTime) {
		t.stopTime = ts
	}
}

func (t *timer) duration() time.Duration {
	return t.stopTime.Sub(t.startTime)
}

func (t *timer) measure() stats.Measure {
	return stats.Measure(t.duration())
}
