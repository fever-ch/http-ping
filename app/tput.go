// Copyright 2022-2023 - Raphaël P. Barazzutti
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

type throughputMeasurer struct {
	ts       time.Time
	counter  uint64
	duration stats.Measure
	logger   *PingLogger
}

type throughputMeasure struct {
	dt              time.Duration
	count           uint64
	queriesDuration stats.Measure
}

func newThroughputMeasurer() *throughputMeasurer {
	return &throughputMeasurer{ts: time.Now()}
}

func (throughputMeasurer *throughputMeasurer) Count(duration stats.Measure) {
	throughputMeasurer.counter++
	throughputMeasurer.duration += duration
}

func (throughputMeasurer *throughputMeasurer) Measure() throughputMeasure {
	now := time.Now()
	dq := throughputMeasurer.counter
	dt := now.Sub(throughputMeasurer.ts)
	qd := throughputMeasurer.duration
	throughputMeasurer.counter = 0
	throughputMeasurer.duration = 0
	throughputMeasurer.ts = now

	return throughputMeasure{
		dt:              dt,
		count:           dq,
		queriesDuration: qd,
	}
}

func (measure *throughputMeasure) String() string {
	x := 1e9 * float64(measure.count) / float64(measure.dt.Nanoseconds())
	return fmt.Sprintf("%.1f", x)
}

func (measure *throughputMeasure) Add(other throughputMeasure) throughputMeasure {
	return throughputMeasure{
		dt:              measure.dt + other.dt,
		count:           measure.count + other.count,
		queriesDuration: measure.queriesDuration + other.queriesDuration,
	}
}

func (measure *throughputMeasure) Sub(other throughputMeasure) throughputMeasure {
	return throughputMeasure{
		dt:              measure.dt - other.dt,
		count:           measure.count - other.count,
		queriesDuration: measure.queriesDuration - other.queriesDuration,
	}
}
