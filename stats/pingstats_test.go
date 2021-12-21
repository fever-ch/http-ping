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
