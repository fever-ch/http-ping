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

package stats

import (
	"math"
	"time"
)

// Some variables for invalid measures. Naming should be explicit enough
const (
	invalid = -int64(1)<<63 + 10

	MeasureNotStarted     = Measure(invalid)
	MeasureNotStopped     = Measure(invalid - 1)
	MeasureNotInitialized = Measure(invalid - 2)
)

// Measure represent a time measurement which can be successful or not
type Measure time.Duration

// IsValid returns true if the measurement is valid (might be negative (!), but is the difference between two time measurements
func (m Measure) IsValid() bool {
	return int64(m) > invalid
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
