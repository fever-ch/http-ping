package stats

import (
	"fmt"
	"math"
	"time"
)

// PingStats represents the statistics which can be computed from an array of latencies
type PingStats struct {
	Average time.Duration
	Min     time.Duration
	Max     time.Duration
	StdDev  time.Duration
}

// PingStatsFromLatencies computes PingStats from a serie of ping measurements.
func PingStatsFromLatencies(measures []time.Duration) *PingStats {
	var sum = time.Duration(0)
	var max = time.Duration(math.MinInt64)
	var min = time.Duration(math.MaxInt64)

	for _, m := range measures {
		sum += m
		if m > max {
			max = m
		}
		if m < min {
			min = m
		}
	}
	average := float64(sum) / float64(len(measures))

	var squaresSum = 0.0

	for _, m := range measures {
		diff := float64(m) - average
		squaresSum += diff * diff
	}
	stdDev := math.Sqrt(squaresSum / float64(len(measures)))

	return &PingStats{
		Min:     min,
		Max:     max,
		Average: time.Duration(average),
		StdDev:  time.Duration(stdDev),
	}
}

func (ps *PingStats) String() string {
	ms := func(d time.Duration) float64 {
		return float64(d.Nanoseconds()) / 1e6
	}

	return fmt.Sprintf("round-trip min/avg/max/stddev = %.3f/%.3f/%.3f/%.3f ms", ms(ps.Min), ms(ps.Average), ms(ps.Max), ms(ps.StdDev))
}
