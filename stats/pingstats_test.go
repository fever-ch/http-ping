package stats

import (
	"testing"
	"time"
)

func TestPingStatsFromLatencies(t *testing.T) {
	input := []Measure{1 * Measure(time.Second), Measure(2 * time.Second), Measure(3 * time.Second)}
	want := "round-trip min/avg/max/stddev = 1000.000/2000.000/3000.000/816.497 ms"
	got := PingStatsFromLatencies(input).String()

	if got != want {
		t.Errorf("Duration was incorrect, got: %s, want: %s.", got, want)
	}
}
