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
