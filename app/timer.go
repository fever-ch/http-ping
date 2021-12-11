package app

import (
	"math"
	"time"
)

type timer struct {
	startTime int64
	stopTime  int64
}

func newTimer() *timer {
	return &timer{
		math.MaxInt64,
		math.MinInt64,
	}
}

func nowTs() int64 {
	return time.Now().UnixMilli()
}

func (t *timer) start() {
	ts := nowTs()
	if ts < t.startTime {
		t.startTime = ts
	}
}

func (t *timer) stop() {
	ts := nowTs()
	if ts > t.stopTime {
		t.stopTime = ts
	}
}

func (t *timer) duration() int64 {
	return t.stopTime - t.startTime
}
