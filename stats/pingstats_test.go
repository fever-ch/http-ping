package stats

import (
	"testing"
	"time"
)

func TestPingStatsFromLatencies(t *testing.T) {
	input := []time.Duration{1 * time.Second, 2 * time.Second, 3 * time.Second}
	want := "round-trip min/avg/max/stddev = 1000.000/2000.000/3000.000/816.497 ms"
	got := PingStatsFromLatencies(input).String()

	if got != want {
		t.Errorf("Duration was incorrect, got: %s, want: %s.", got, want)
	}
}
