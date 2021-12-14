package stats

import "time"

const (
	invalid = 1 << 63

	//MeasureNotStarted = Measure(invalid + 1)
	//MeasureNotStopped = Measure(invalid + 2)
)

type Measure time.Duration

func (m Measure) isValid() bool {
	return true
	//return int64(m) > int64(MeasureNotStopped)
}

func (m Measure) isSuccess() bool {
	return int64(m) > 0
}

func (m Measure) GetDuration() (duration time.Duration, success bool) {
	if m.isSuccess() {
		return time.Duration(m), true
	}
	return time.Duration(-m), false

}
