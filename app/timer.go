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

func (t *timer) toMeasure() stats.Measure {
	if t.startTime != defaultStartTime && t.stopTime != defaultStopTime {
		return stats.Measure(t.stopTime.Sub(t.startTime))
	} else if t.startTime == defaultStartTime {
		if t.stopTime == defaultStopTime {
			return stats.MeasureNotInitialized
		}
		return stats.MeasureNotStarted
	} else {
		return stats.MeasureNotStopped
	}
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
