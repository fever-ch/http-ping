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
	"fmt"
	"time"
)

// PingStats represents the statistics which can be computed from an array of latencies
type PingStats struct {
	Average Measure
	Min     Measure
	Max     Measure
	StdDev  Measure
}

// PingStatsFromLatencies computes PingStats from a serie of ping measurements.
func PingStatsFromLatencies(measures []Measure) *PingStats {

	stats := ComputeStats(measuresIterable(measures))

	return &PingStats{
		Min:     Measure(stats.Min),
		Max:     Measure(stats.Max),
		Average: Measure(stats.Average),
		StdDev:  Measure(stats.StdDev),
	}
}

func (ps *PingStats) String() string {
	ms := func(d Measure) float64 {
		return d.ToFloat(time.Millisecond)
	}

	return fmt.Sprintf("round-trip min/avg/max/stddev = %.3f/%.3f/%.3f/%.3f ms", ms(ps.Min), ms(ps.Average), ms(ps.Max), ms(ps.StdDev))
}

type measuresIterable []Measure

func (m measuresIterable) Iterator() Iterator {
	return &measuresIterator{measures: m}
}

type measuresIterator struct {
	measures []Measure
	nextPos  int
}

func (m *measuresIterator) HasNext() bool {
	return m.nextPos < len(m.measures)
}

func (m *measuresIterator) Next() Observation {
	val := Observation{Value: m.measures[m.nextPos].ToFloat(time.Nanosecond), Weight: 1.0}
	m.nextPos++
	return val
}
