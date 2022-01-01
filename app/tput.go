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

package app

import (
	"fever.ch/http-ping/stats"
	"fmt"
	"time"
)

type tputMeasurer struct {
	ts       time.Time
	counter  uint64
	duration stats.Measure
}

type tputMeasure struct {
	dt              time.Duration
	count           uint64
	queriesDuration stats.Measure
}

func (measure *tputMeasure) averageLatency(unit time.Duration) float64 {
	return measure.queriesDuration.ToFloat(unit) / float64(measure.count)
}

func newTputMeasurer() *tputMeasurer {
	return &tputMeasurer{ts: time.Now()}
}

func (tputMeasurer *tputMeasurer) Count(duration stats.Measure) {
	tputMeasurer.counter++
	tputMeasurer.duration += duration
}

func (tputMeasurer *tputMeasurer) Measure() tputMeasure {
	now := time.Now()
	dq := tputMeasurer.counter
	dt := now.Sub(tputMeasurer.ts)
	qd := tputMeasurer.duration
	tputMeasurer.counter = 0
	tputMeasurer.duration = 0
	tputMeasurer.ts = now

	return tputMeasure{
		dt:              dt,
		count:           dq,
		queriesDuration: qd,
	}
}

func (measure *tputMeasure) String() string {
	x := 1e9 * float64(measure.count) / float64(measure.dt.Nanoseconds())
	return fmt.Sprintf("%.1f", x)
}

func (measure *tputMeasure) Add(other tputMeasure) tputMeasure {
	return tputMeasure{
		dt:              measure.dt + other.dt,
		count:           measure.count + other.count,
		queriesDuration: measure.queriesDuration + other.queriesDuration,
	}
}

func (measure *tputMeasure) Sub(other tputMeasure) tputMeasure {
	return tputMeasure{
		dt:              measure.dt - other.dt,
		count:           measure.count - other.count,
		queriesDuration: measure.queriesDuration - other.queriesDuration,
	}
}
