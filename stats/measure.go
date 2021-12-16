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
