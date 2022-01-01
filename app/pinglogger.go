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
	onMeasure(httpMeasure *HTTPMeasure, id int)
	onTick(measure tputMeasure)
	onClose(attempts int64, success int64, lossRate float64, pingStats *stats.PingStats)
}

type baseLogger struct {
	config *Config
	stdout io.Writer
	pinger Pinger
}

type quietLogger baseLogger

func newQuietLogger(config *Config, stdout io.Writer, pinger Pinger) logger {
	return &quietLogger{config: config, stdout: stdout, pinger: pinger}
}

func (quietLogger *quietLogger) onMeasure(_ *HTTPMeasure, _ int) {

}

func (quietLogger *quietLogger) onTick(_ tputMeasure) {

}

func (quietLogger *quietLogger) onClose(attempts int64, successes int64, lossRate float64, pingStats *stats.PingStats) {

	_, _ = fmt.Fprintf(quietLogger.stdout, "--- %s ping statistics ---\n", quietLogger.pinger.URL())

	_, _ = fmt.Fprintf(quietLogger.stdout, "%d requests sent, %d answers received, %.1f%% loss\n", attempts, successes, lossRate)

	if successes > 0 {
		_, _ = fmt.Fprintf(quietLogger.stdout, "%s\n", pingStats.String())
	}
}

//type standardLogger quietLogger
type standardLogger struct {
	quietLogger
}

func newStandardLogger(config *Config, stdout io.Writer, pinger Pinger) *standardLogger {
	return &standardLogger{quietLogger{config: config, stdout: stdout, pinger: pinger}}
}

func (logger *standardLogger) onMeasure(measure *HTTPMeasure, id int) {
	if logger.config.Tput {
		return
	}
	if measure.IsFailure {
		_, _ = fmt.Fprintf(logger.stdout, "%4d: Error: %s\n", id, measure.FailureCause)
		return
	}
	_, _ = fmt.Fprintf(logger.stdout, "%8d: %s, code=%d, size=%d bytes, time=%.1f ms\n", id, measure.RemoteAddr, measure.StatusCode, measure.Bytes, measure.TotalTime.ToFloat(time.Millisecond))

}

func (logger *standardLogger) onTick(m tputMeasure) {

	fmt.Printf("          throughput: %s queries/sec, average latency: %.1f ms\n", m.String(), m.queriesDuration.ToFloat(time.Microsecond)/float64(1000*m.count))
}

func (logger *standardLogger) onClose(attempts int64, successes int64, lossRate float64, pingStats *stats.PingStats) {
	_, _ = fmt.Fprintf(logger.stdout, "\n")
	_, _ = fmt.Fprintf(logger.stdout, "--- %s ping statistics ---\n", logger.pinger.URL())

	_, _ = fmt.Fprintf(logger.stdout, "%d requests sent, %d answers received, %.1f%% loss\n", attempts, successes, lossRate)

	if successes > 0 {
		_, _ = fmt.Fprintf(logger.stdout, "%s\n", pingStats.String())
	}
}

type verboseLogger struct {
	standardLogger
	measureSum *HTTPMeasure
}

func newVerboseLogger(config *Config, stdout io.Writer, pinger Pinger) logger {
	return &verboseLogger{
		standardLogger: *newStandardLogger(config, stdout, pinger),
		measureSum: &HTTPMeasure{
			DNSResolution: stats.MeasureNotValid,
			TCPHandshake:  stats.MeasureNotValid,
			TLSDuration:   stats.MeasureNotValid,
		},
	}
}

func (verboseLogger *verboseLogger) onMeasure(measure *HTTPMeasure, id int) {
	if verboseLogger.config.Tput {
		return
	}
	if measure.IsFailure {
		_, _ = fmt.Fprintf(verboseLogger.stdout, "%4d: Error: %s\n", id, measure.FailureCause)
		return
	}

	_, _ = fmt.Fprintf(verboseLogger.stdout, "%8d: %s, code=%d, size=%d bytes, time=%.1f ms\n", id, measure.RemoteAddr, measure.StatusCode, measure.Bytes, measure.TotalTime.ToFloat(time.Millisecond))
	_, _ = fmt.Fprintf(verboseLogger.stdout, "          proto=%s, socket reused=%t, compressed=%t\n", measure.Proto, measure.SocketReused, measure.Compressed)
	_, _ = fmt.Fprintf(verboseLogger.stdout, "          network i/o: bytes read=%d, bytes written=%d\n", measure.InBytes, measure.OutBytes)

	if measure.TLSEnabled {
		_, _ = fmt.Fprintf(verboseLogger.stdout, "          tls version=%s\n", measure.TLSVersion)
	}

	verboseLogger.measureSum.TotalTime += measure.TotalTime

	verboseLogger.measureSum.ConnEstablishment = verboseLogger.measureSum.ConnEstablishment.SumIfValid(measure.ConnEstablishment)
	verboseLogger.measureSum.DNSResolution = verboseLogger.measureSum.DNSResolution.SumIfValid(measure.DNSResolution)
	verboseLogger.measureSum.TCPHandshake = verboseLogger.measureSum.TCPHandshake.SumIfValid(measure.TCPHandshake)
	verboseLogger.measureSum.TLSDuration = verboseLogger.measureSum.TLSDuration.SumIfValid(measure.TLSDuration)
	verboseLogger.measureSum.RequestSending += measure.RequestSending
	verboseLogger.measureSum.Wait += measure.Wait
	verboseLogger.measureSum.ResponseIngesting += measure.ResponseIngesting

	_, _ = fmt.Fprintf(verboseLogger.stdout, "\n")

	_, _ = fmt.Fprintf(verboseLogger.stdout, "          latency contributions:\n")

	verboseLogger.drawMeasure(measure, verboseLogger.stdout)

	_, _ = fmt.Fprintf(verboseLogger.stdout, "\n")
}

func (verboseLogger *verboseLogger) onClose(attempts int64, successes int64, lossRate float64, pingStats *stats.PingStats) {
	_, _ = fmt.Fprintf(verboseLogger.stdout, "\n")
	_, _ = fmt.Fprintf(verboseLogger.stdout, "--- %s ping statistics ---\n", verboseLogger.pinger.URL())

	_, _ = fmt.Fprintf(verboseLogger.stdout, "%d requests sent, %d answers received, %.1f%% loss\n", attempts, successes, lossRate)

	if successes > 0 {
		_, _ = fmt.Fprintf(verboseLogger.stdout, "%s\n", pingStats.String())

		verboseLogger.measureSum.TotalTime = verboseLogger.measureSum.TotalTime.Divide(successes)
		verboseLogger.measureSum.ConnEstablishment = verboseLogger.measureSum.ConnEstablishment.Divide(successes)
		verboseLogger.measureSum.DNSResolution = verboseLogger.measureSum.DNSResolution.Divide(successes)
		verboseLogger.measureSum.TCPHandshake = verboseLogger.measureSum.TCPHandshake.Divide(successes)
		verboseLogger.measureSum.TLSDuration = verboseLogger.measureSum.TLSDuration.Divide(successes)
		verboseLogger.measureSum.RequestSending = verboseLogger.measureSum.RequestSending.Divide(successes)
		verboseLogger.measureSum.Wait = verboseLogger.measureSum.Wait.Divide(successes)
		verboseLogger.measureSum.ResponseIngesting = verboseLogger.measureSum.ResponseIngesting.Divide(successes)

		verboseLogger.measureSum.TLSEnabled = verboseLogger.measureSum.TLSDuration > 0

		_, _ = fmt.Fprintf(verboseLogger.stdout, "\naverage latency contributions:\n")

		verboseLogger.drawMeasure(verboseLogger.measureSum, verboseLogger.stdout)
	}
}

func (verboseLogger *verboseLogger) drawMeasure(measure *HTTPMeasure, stdout io.Writer) {
	entries := measureEntry{
		label:    "request and response",
		duration: measure.TotalTime,
		children: []*measureEntry{
			{label: "connection setup", duration: measure.ConnEstablishment,
				children: []*measureEntry{
					{label: "DNS resolution", duration: measure.DNSResolution},
					{label: "TCP handshake", duration: measure.TCPHandshake},
					{label: "TLS handshake", duration: measure.TLSDuration},
				}},
			{label: "request sending", duration: measure.RequestSending},
			{label: "wait", duration: measure.Wait},
			{label: "response ingestion", duration: measure.ResponseIngesting},
		},
	}
	if !measure.TLSEnabled {
		entries.children[0].children = entries.children[0].children[0:2]
	}

	l := verboseLogger.makeTreeList(&entries)

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

func (verboseLogger *verboseLogger) makeTreeList(root *measureEntry) []measureEntryVisit {
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
