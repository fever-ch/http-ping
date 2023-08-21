// Copyright 2022-2023 - Raphaël P. Barazzutti
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

package app

import (
	"fever.ch/http-ping/stats"
	"strings"
	"time"
)

type PingLogger interface {
	onMeasure(httpMeasure *HTTPMeasure)
	onTick(measure throughputMeasure)
	onClose()
	onThroughputClose()
	bell()
	Printf(format string, a ...any) (int, error)
}

type measures struct {
	successes int64
	attempts  int64
	latencies []stats.Measure
}

type quietLogger struct {
	config             *Config
	consoleLogger      ConsoleLogger
	pinger             Pinger
	measures           measures
	throughputMeasures []throughputMeasure
}

func newQuietLogger(config *Config, consoleLogger ConsoleLogger, pinger Pinger) PingLogger {
	return &quietLogger{config: config, consoleLogger: consoleLogger, pinger: pinger}
}

func (logger *quietLogger) Printf(format string, a ...any) (int, error) {
	return logger.consoleLogger.Printf(format, a...)
}

func (logger *quietLogger) onMeasure(m *HTTPMeasure) {
	logger.measures.attempts++
	if !m.IsFailure {
		logger.measures.successes++
		logger.measures.latencies = append(logger.measures.latencies, m.MeasuresCollection.Get(stats.Total))
	}
}

type throughputMeasuresIterable []throughputMeasure

func (m throughputMeasuresIterable) Iterator() stats.Iterator {
	return &throughputMeasuresIterator{measures: m}
}

type throughputMeasuresIterator struct {
	measures []throughputMeasure
	nextPos  int
}

func (m *throughputMeasuresIterator) HasNext() bool {
	return m.nextPos < len(m.measures)
}

func (m *throughputMeasuresIterator) Next() stats.Observation {
	cur := m.measures[m.nextPos]
	dt := float64(cur.dt) / float64(time.Second)
	val := stats.Observation{Value: float64(cur.count) / dt, Weight: dt}
	m.nextPos++
	return val
}

func (logger *quietLogger) onTick(m throughputMeasure) {
	logger.throughputMeasures = append(logger.throughputMeasures, m)
}

func (logger *quietLogger) onClose() {
	var lossRate = float64(0)
	if logger.measures.attempts > 0 {
		lossRate = float64(logger.measures.attempts-logger.measures.successes) / float64(logger.measures.attempts)
	}
	pingStats := stats.PingStatsFromLatencies(logger.measures.latencies)

	_, _ = logger.Printf("--- %s ping statistics ---\n", logger.pinger.URL())

	_, _ = logger.Printf("%d requests sent, %d answers received, %.1f%% loss\n", logger.measures.attempts, logger.measures.successes, lossRate*100)

	if logger.measures.successes > 0 {
		_, _ = logger.Printf("%s\n", pingStats.String())
	}
}

func (logger *quietLogger) onThroughputClose() {
	_, _ = logger.Printf("\n")
	if len(logger.throughputMeasures) > 0 {
		stat := stats.ComputeStats(throughputMeasuresIterable(logger.throughputMeasures))
		_, _ = logger.Printf("throughput measures:\n")
		_, _ = logger.Printf("queries throughput min/avg/max/stdev = %.1f/%.1f/%.1f/%.1f queries/sec \n", stat.Min, stat.Average, stat.Max, stat.StdDev)
	} else {
		_, _ = logger.Printf("not enough time to collect data\n")
	}
}

type standardLogger struct {
	quietLogger
}

func newStandardLogger(config *Config, consoleLogger ConsoleLogger, pinger Pinger) *standardLogger {
	return &standardLogger{quietLogger{config: config, consoleLogger: consoleLogger, pinger: pinger}}
}

func (logger *standardLogger) onMeasure(measure *HTTPMeasure) {
	logger.quietLogger.onMeasure(measure)

	if logger.config.Throughput {
		return
	}
	if measure.IsFailure {
		_, _ = logger.Printf("%4d: Error: %s\n", logger.measures.attempts, measure.FailureCause)
		return
	}
	_, _ = logger.Printf("%8d: %s, %s, code=%d, size=%d bytes, time=%.1f ms\n", logger.measures.attempts, measure.Proto, measure.RemoteAddr, measure.StatusCode, measure.Bytes, measure.MeasuresCollection.Get(stats.Total).ToFloat(time.Millisecond))
}

func (logger *standardLogger) onTick(m throughputMeasure) {
	logger.quietLogger.onTick(m)
	logger.Printf("          throughput: %s queries/sec, average latency: %.1f ms\n", m.String(), m.queriesDuration.ToFloat(time.Microsecond)/float64(1000*m.count))
}

func (logger *standardLogger) onClose() {
	_, _ = logger.Printf("\n")

	logger.quietLogger.onClose()
}

type verboseLogger struct {
	standardLogger
	measureSum *HTTPMeasure
}

func (logger *quietLogger) bell() {
	if logger.config.AudibleBell {
		_, _ = logger.Printf("\a")
	}
}

func newVerboseLogger(config *Config, consoleLogger ConsoleLogger, pinger Pinger) PingLogger {
	return &verboseLogger{
		standardLogger: *newStandardLogger(config, consoleLogger, pinger),
		measureSum: &HTTPMeasure{
			MeasuresCollection: stats.NewMeasureRegistry(),
		},
	}
}

func (logger *verboseLogger) onMeasure(measure *HTTPMeasure) {
	if strings.HasPrefix(measure.Proto, "HTTP/3") {
		measure.MeasuresCollection.Set(stats.QUIC, measure.MeasuresCollection.Get(stats.TLS))
		measure.MeasuresCollection.Set(stats.TLS, stats.MeasureNotValid)
	} else {
		measure.MeasuresCollection.Set(stats.ReqAndWait, stats.MeasureNotValid)
	}
	logger.standardLogger.onMeasure(measure)
	if logger.config.Throughput {
		return
	}
	if measure.IsFailure {
		return
	}

	_, _ = logger.Printf("          proto=%s, socket reused=%t, compressed=%t\n", measure.Proto, measure.SocketReused, measure.Compressed)
	_, _ = logger.Printf("          network i/o: bytes read=%d, bytes written=%d\n", measure.InBytes, measure.OutBytes)

	_, _ = logger.Printf("          tls version=%s\n", measure.TLSVersion)
	logger.measureSum.MeasuresCollection.Append(measure.MeasuresCollection)

	_, _ = logger.Printf("\n\n")

	_, _ = logger.Printf("          latency contributions:\n")

	logger.drawMeasure(measure)

	_, _ = logger.Printf("\n")
}

func (logger *verboseLogger) onClose() {

	logger.standardLogger.onClose()
	successes := logger.measures.successes

	if successes > 0 && !logger.config.Throughput {
		logger.measureSum.MeasuresCollection.Divide(successes)

		_, _ = logger.Printf("\naverage latency contributions:\n")

		logger.drawMeasure(logger.measureSum)
	}
}

func (logger *verboseLogger) drawMeasure(measure *HTTPMeasure) {
	entries := measureEntry{
		label:    "request and response",
		duration: measure.MeasuresCollection.Get(stats.Total),
		children: []*measureEntry{
			{label: "connection setup", duration: measure.MeasuresCollection.Get(stats.Conn),
				children: []*measureEntry{
					{label: "DNS resolution", duration: measure.MeasuresCollection.Get(stats.DNS)},
					{label: "TCP handshake", duration: measure.MeasuresCollection.Get(stats.TCP)},
					{label: "QUIC handshake", duration: measure.MeasuresCollection.Get(stats.QUIC)},
					{label: "TLS handshake", duration: measure.MeasuresCollection.Get(stats.TLS)},
				}},
			{label: "request sending and wait for answer", duration: measure.MeasuresCollection.Get(stats.ReqAndWait)},
			{label: "request sending", duration: measure.MeasuresCollection.Get(stats.Req)},
			{label: "wait", duration: measure.MeasuresCollection.Get(stats.Wait)},
			{label: "response ingestion", duration: measure.MeasuresCollection.Get(stats.Resp)},
		},
	}

	l := logger.makeTreeList(&entries)

	for i, e := range l {
		pipes := make([]string, e.depth)
		for j := 0; j < e.depth; j++ {
			if i+1 >= len(l) || l[i+1].depth-1 < j {
				pipes[j] = " └─"
			} else if j == e.depth-1 {
				pipes[j] = " ├─"
			} else {
				pipes[j] = " │ "
			}

		}
		_, _ = logger.Printf("          ")
		for i := 0; i < e.depth; i++ {
			_, _ = logger.Printf("          %s ", pipes[i])
		}

		_, _ = logger.Printf("%6.1f ms %s\n", e.measureEntry.duration.ToFloat(time.Millisecond), e.measureEntry.label)
	}
}

type measureEntry struct {
	label    string
	duration stats.Measure
	children []*measureEntry
}

type measureEntryVisit struct {
	measureEntry *measureEntry
	depth        int
}

func (logger *verboseLogger) makeTreeList(root *measureEntry) []measureEntryVisit {
	var list []measureEntryVisit

	var visit func(entry *measureEntry, depth int)

	visit = func(entry *measureEntry, depth int) {
		if entry.duration.IsValid() {
			list = append(list, measureEntryVisit{entry, depth})
		}

		for _, e := range entry.children {
			visit(e, depth+1)
		}

	}

	visit(root, 0)

	return list
}
