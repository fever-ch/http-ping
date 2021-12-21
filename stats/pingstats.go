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
	"fmt"
	"math"
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
	var sum = Measure(0)
	var max = Measure(math.MinInt64)
	var min = Measure(math.MaxInt64)

	count := 0
	for _, m := range measures {
		if m.IsSuccess() {
			count++
			sum += m
			if m > max {
				max = m
			}
			if m < min {
				min = m
			}
		}
	}
	average := float64(sum) / float64(count)

	var squaresSum = 0.0

	for _, m := range measures {
		diff := float64(m) - average
		squaresSum += diff * diff
	}
	stdDev := math.Sqrt(squaresSum / float64(count))

	return &PingStats{
		Min:     min,
		Max:     max,
		Average: Measure(average),
		StdDev:  Measure(stdDev),
	}
}

func (ps *PingStats) String() string {
	ms := func(d Measure) float64 {
		return d.ToFloat(time.Millisecond)
	}

	return fmt.Sprintf("round-trip min/avg/max/stddev = %.3f/%.3f/%.3f/%.3f ms", ms(ps.Min), ms(ps.Average), ms(ps.Max), ms(ps.StdDev))
}
