// Copyright 2021 Raphaël P. Barazzutti
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

package app

import (
	"fever.ch/http-ping/stats"
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
	if t.startTime == defaultStartTime || t.stopTime == defaultStopTime {
		return stats.MeasureNotInitialized
	}
	return stats.Measure(t.duration())
}
