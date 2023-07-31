// Copyright 2022 Raphaël P. Barazzutti
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package stats

import (
	"math"
	"time"
)

var (
	defaultStartTime = time.UnixMicro(math.MaxInt64)
	defaultStopTime  = time.UnixMicro(math.MinInt64)
)

type TimerType int

type Timer struct {
	startTime, stopTime time.Time
}

func NewTimer() *Timer {
	return &Timer{
		defaultStartTime,
		defaultStopTime,
	}
}

func (t *Timer) Start() {
	ts := time.Now()
	if ts.Before(t.startTime) {
		t.startTime = ts
	}
}

func (t *Timer) Stop() {
	ts := time.Now()
	if ts.After(t.stopTime) {
		t.stopTime = ts
	}
}

func (t *Timer) Duration() time.Duration {
	return t.stopTime.Sub(t.startTime)
}

func (t *Timer) measure() Measure {
	if t.startTime == defaultStartTime || t.stopTime == defaultStopTime {
		return MeasureNotInitialized
	}
	return Measure(t.Duration())
}

//totalTimer := NewTimer()
//connTimer := NewTimer()
//dnsTimer := NewTimer()
//tlsTimer := NewTimer()
//tcpTimer := NewTimer()
//reqTimer := NewTimer()
//waitTimer := NewTimer()
//responseTimer := NewTimer()

const (
	Total TimerType = iota
	Conn
	DNS
	TLS
	PreQUIC
	FullQUIC
	TCP
	Req
	Wait
	Resp
)

type TimerRegistry struct {
	timers map[TimerType]*Timer
}

func NewTimerRegistry() *TimerRegistry {
	return &TimerRegistry{
		timers: make(map[TimerType]*Timer),
	}
}

func (tr *TimerRegistry) Get(timerType TimerType) *Timer {
	if _, ok := tr.timers[timerType]; !ok {
		tr.timers[timerType] = NewTimer()
	}
	value, _ := tr.timers[timerType]
	return value
}

func (tr *TimerRegistry) Measure() *MeasureRegistry {
	mr := NewMeasureRegistry()
	for k, v := range tr.timers {
		mr.timers[k] = v.measure()
	}
	return mr
}
