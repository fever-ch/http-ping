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

package app

import (
	"fever.ch/http-ping/stats"
	"fmt"
	"io"
	"time"
)

type logger interface {
	onMeasure(httpMeasure *HTTPMeasure)
	onTick(measure throughputMeasure)
	onClose()
	onThroughputClose()
}

type measures struct {
	successes int64
	attempts  int64
	latencies []stats.Measure
}

type quietLogger struct {
	config             *Config
	stdout             io.Writer
	pinger             Pinger
	measures           measures
	throughputMeasures []throughputMeasure
}

func newQuietLogger(config *Config, stdout io.Writer, pinger Pinger) logger {
	return &quietLogger{config: config, stdout: stdout, pinger: pinger}
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

	_, _ = fmt.Fprintf(logger.stdout, "--- %s ping statistics ---\n", logger.pinger.URL())

	_, _ = fmt.Fprintf(logger.stdout, "%d requests sent, %d answers received, %.1f%% loss\n", logger.measures.attempts, logger.measures.successes, lossRate*100)

	if logger.measures.successes > 0 {
		_, _ = fmt.Fprintf(logger.stdout, "%s\n", pingStats.String())
	}
}

func (logger *quietLogger) onThroughputClose() {
	_, _ = fmt.Fprintf(logger.stdout, "\n")
	if len(logger.throughputMeasures) > 0 {
		stat := stats.ComputeStats(throughputMeasuresIterable(logger.throughputMeasures))
		_, _ = fmt.Fprintf(logger.stdout, "throughput measures:\n")
		_, _ = fmt.Fprintf(logger.stdout, "queries throughput min/avg/max/stdev = %.1f/%.1f/%.1f/%.1f queries/sec \n", stat.Min, stat.Average, stat.Max, stat.StdDev)
	} else {
		_, _ = fmt.Fprintf(logger.stdout, "not enough time to collect data\n")
	}
}

type standardLogger struct {
	quietLogger
}

func newStandardLogger(config *Config, stdout io.Writer, pinger Pinger) *standardLogger {
	return &standardLogger{quietLogger{config: config, stdout: stdout, pinger: pinger}}
}

func (logger *standardLogger) onMeasure(measure *HTTPMeasure) {
	logger.quietLogger.onMeasure(measure)

	if logger.config.Throughput {
		return
	}
	if measure.IsFailure {
		_, _ = fmt.Fprintf(logger.stdout, "%4d: Error: %s\n", logger.measures.attempts, measure.FailureCause)
		return
	}
	_, _ = fmt.Fprintf(logger.stdout, "%8d: %s, %s, code=%d, size=%d bytes, time=%.1f ms\n", logger.measures.attempts, measure.Proto, measure.RemoteAddr, measure.StatusCode, measure.Bytes, measure.MeasuresCollection.Get(stats.Total).ToFloat(time.Millisecond))
}

func (logger *standardLogger) onTick(m throughputMeasure) {
	logger.quietLogger.onTick(m)
	fmt.Printf("          throughput: %s queries/sec, average latency: %.1f ms\n", m.String(), m.queriesDuration.ToFloat(time.Microsecond)/float64(1000*m.count))
}

func (logger *standardLogger) onClose() {
	_, _ = fmt.Fprintf(logger.stdout, "\n")

	logger.quietLogger.onClose()
}

type verboseLogger struct {
	standardLogger
	//stats.MeasuresCollection
	measureSum *HTTPMeasure
}

func newVerboseLogger(config *Config, stdout io.Writer, pinger Pinger) logger {
	return &verboseLogger{
		standardLogger: *newStandardLogger(config, stdout, pinger),
		measureSum: &HTTPMeasure{
			MeasuresCollection: stats.NewMeasureRegistry(),
			//DNSResolution: stats.MeasureNotValid,
			//TCPHandshake:  stats.MeasureNotValid,
			//TLSDuration:   stats.MeasureNotValid,
		},
	}
}

func (logger *verboseLogger) onMeasure(measure *HTTPMeasure) {
	logger.standardLogger.onMeasure(measure)
	if logger.config.Throughput {
		return
	}
	if measure.IsFailure {
		return
	}

	_, _ = fmt.Fprintf(logger.stdout, "          proto=%s, socket reused=%t, compressed=%t\n", measure.Proto, measure.SocketReused, measure.Compressed)
	_, _ = fmt.Fprintf(logger.stdout, "          network i/o: bytes read=%d, bytes written=%d\n", measure.InBytes, measure.OutBytes)

	//if measure.TLSEnabled {
	_, _ = fmt.Fprintf(logger.stdout, "          tls version=%s\n", measure.TLSVersion)
	//}
	//logger.
	logger.measureSum.MeasuresCollection.Append(measure.MeasuresCollection)
	//logger.measureSum.TotalTime += measure.TotalTime
	//logger.measureSum.ConnEstablishment = logger.measureSum.ConnEstablishment.SumIfValid(measure.ConnEstablishment)
	//logger.measureSum.DNSResolution = logger.measureSum.DNSResolution.SumIfValid(measure.DNSResolution)
	//logger.measureSum.TCPHandshake = logger.measureSum.TCPHandshake.SumIfValid(measure.TCPHandshake)
	//logger.measureSum.TLSDuration = logger.measureSum.TLSDuration.SumIfValid(measure.TLSDuration)
	//logger.measureSum.RequestSending += measure.RequestSending
	//logger.measureSum.Wait += measure.Wait
	//logger.measureSum.ResponseIngesting += measure.ResponseIngesting

	_, _ = fmt.Fprintf(logger.stdout, "\n")

	_, _ = fmt.Fprintf(logger.stdout, "          latency contributions:\n")

	logger.drawMeasure(measure, logger.stdout)

	_, _ = fmt.Fprintf(logger.stdout, "\n")
}

func (logger *verboseLogger) onClose() {

	logger.standardLogger.onClose()
	successes := logger.measures.successes

	if successes > 0 && !logger.config.Throughput {
		logger.measureSum.MeasuresCollection.Divide(successes)
		//logger.measureSum.TotalTime = logger.measureSum.TotalTime.Divide(successes)
		//logger.measureSum.ConnEstablishment = logger.measureSum.ConnEstablishment.Divide(successes)
		//logger.measureSum.DNSResolution = logger.measureSum.DNSResolution.Divide(successes)
		//logger.measureSum.TCPHandshake = logger.measureSum.TCPHandshake.Divide(successes)
		//logger.measureSum.TLSDuration = logger.measureSum.TLSDuration.Divide(successes)
		//logger.measureSum.RequestSending = logger.measureSum.RequestSending.Divide(successes)
		//logger.measureSum.Wait = logger.measureSum.Wait.Divide(successes)
		//logger.measureSum.ResponseIngesting = logger.measureSum.ResponseIngesting.Divide(successes)
		//logger.measureSum.TLSEnabled = logger.measureSum.MeasuresCollection.Get(stats.TLS) > 0
		//logger.measureSum.TLSEnabled = logger.measureSum.TLSDuration > 0

		_, _ = fmt.Fprintf(logger.stdout, "\naverage latency contributions:\n")

		logger.drawMeasure(logger.measureSum, logger.stdout)
	}
}

func (logger *verboseLogger) drawMeasure(measure *HTTPMeasure, stdout io.Writer) {
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
			{label: "request sending", duration: measure.MeasuresCollection.Get(stats.Req)},
			{label: "wait", duration: measure.MeasuresCollection.Get(stats.Wait)},
			{label: "response ingestion", duration: measure.MeasuresCollection.Get(stats.Resp)},
		},
	}
	//if !measure.TLSEnabled {
	//	entries.children[0].children = entries.children[0].children[0:2]
	//}

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
		_, _ = fmt.Fprintf(stdout, "          ")
		for i := 0; i < e.depth; i++ {
			_, _ = fmt.Fprintf(stdout, "          %s ", pipes[i])
		}

		_, _ = fmt.Fprintf(stdout, "%6.1f ms %s\n", e.measureEntry.duration.ToFloat(time.Millisecond), e.measureEntry.label)
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
