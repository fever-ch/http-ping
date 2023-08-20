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

// Some variables for invalid measures. Naming should be explicit enough
const (
	invalid = -int64(1)<<63 + 10

	MeasureNotValid       = Measure(invalid)
	MeasureNotInitialized = Measure(invalid - 1)
)

// Measure represent a time measurement which can be successful or not
type Measure time.Duration

// IsValid returns true if the measurement is valid (might be negative (!), but is the difference between two time measurements
func (m Measure) IsValid() bool {
	return int64(m) > invalid
}

// SumIfValid returns
// - if both measures are valid: the sum of them
// - if one of them is invalid, it returns this specific one
// - otherwise it returns an invalid  measure
func (m Measure) SumIfValid(o Measure) Measure {
	if m.IsValid() && o.IsValid() {
		return m + o
	} else if m.IsValid() && !o.IsValid() {
		return m
	} else if !m.IsValid() && o.IsValid() {
		return o
	} else {
		return MeasureNotValid
	}
}

// Divide returns the result of the division of a measure with n
// if the measure is invalid the returned value is invalid as well
func (m Measure) Divide(n int64) Measure {
	if !m.IsValid() {
		return m
	}
	return Measure(int64(m) / n)
}

// IsSuccess returns true if the measurement is valid and positive.
func (m Measure) IsSuccess() bool {
	return int64(m) >= 0
}

// ToFloat converts the current measurement to a float in a specified unit
func (m Measure) ToFloat(unit time.Duration) float64 {
	if !m.IsValid() {
		return math.NaN()
	}
	return float64(time.Duration(m).Nanoseconds()) / float64(unit)
}

func NewMeasureRegistry() *MeasuresCollection {
	return &MeasuresCollection{
		timers: make(map[TimerType]Measure),
	}
}

type MeasuresCollection struct {
	timers map[TimerType]Measure
}

func (mr *MeasuresCollection) Append(other *MeasuresCollection) {
	for r, i := range other.timers {
		if i.IsValid() {
			if v := mr.Get(r); v.IsValid() {
				mr.Set(r, v+i)
			} else {
				mr.Set(r, i)
			}
		}
	}
}

func (mr *MeasuresCollection) Divide(successes int64) {
	for r, i := range mr.timers {
		mr.Set(r, i/Measure(successes))
	}
}

func (mr *MeasuresCollection) Get(tt TimerType) Measure {
	if mr.timers == nil {
		mr.timers = make(map[TimerType]Measure)
	}
	if a, b := mr.timers[tt]; b {
		return a
	}
	return MeasureNotValid
}

func (mr *MeasuresCollection) Set(tt TimerType, m Measure) {
	if mr.timers == nil {
		mr.timers = make(map[TimerType]Measure)
	}
	mr.timers[tt] = m
}
