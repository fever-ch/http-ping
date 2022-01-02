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

import "math"

// Observation represents an item in the population we would extract statistics
type Observation struct {
	Value  float64
	Weight float64
}

// Iterator is used to iterate over Observation
type Iterator interface {
	HasNext() bool
	Next() Observation
}

// Iterable is used to define a collection providing an Iterator
type Iterable interface {
	Iterator() Iterator
}

// Stats is the type returned by ComputeStats
type Stats struct {
	Average float64
	Min     float64
	Max     float64
	StdDev  float64
}

// ComputeStats return statistical information about collection as Stats instance
func ComputeStats(collection Iterable) *Stats {
	min := math.MaxFloat64
	max := -math.MaxFloat64

	totalWeight := 0.0
	total := 0.0

	it := collection.Iterator()
	for it.HasNext() {
		cur := it.Next()

		if cur.Value > max {
			max = cur.Value
		}
		if cur.Value < min {
			min = cur.Value
		}

		totalWeight += cur.Weight
		total += cur.Value
	}

	average := total / totalWeight

	it = collection.Iterator()

	sumDiff := 0.0

	for it.HasNext() {
		cur := it.Next()

		diff := cur.Value - average

		sumDiff += cur.Weight * (diff * diff)
	}
	stdDev := math.Sqrt(sumDiff / totalWeight)

	return &Stats{
		Min:     min,
		Max:     max,
		Average: average,
		StdDev:  stdDev,
	}
}
