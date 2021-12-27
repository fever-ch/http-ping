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
	"os"
	"os/signal"
	"time"
)

// HTTPPing is the main class of this app, now it contains mostly UI logic
type HTTPPing interface {
	Run() error
}

type httpPingImpl struct {
	config *Config
	stdout io.Writer
	pinger Pinger
	logger logger
}

// NewHTTPPing builds a new instance of HTTPPing or error if something goes wrong
func NewHTTPPing(config *Config, stdout io.Writer) (HTTPPing, error) {

	runtimeConfig := &RuntimeConfig{
		RedirectCallBack: func(url string) {
			_, _ = fmt.Fprintf(stdout, "   ─→     Redirected to %s\n\n", url)
		},
	}

	pinger, err := NewPinger(config, runtimeConfig)

	if err != nil {
		return nil, err
	}

	var logger logger

	if config.LogLevel == 0 {
		logger = newQuietLogger(config, stdout, pinger)
	} else if config.LogLevel == 2 {
		logger = newVerboseLogger(config, stdout, pinger)
	} else {
		logger = newStandardLogger(config, stdout, pinger)
	}

	return &httpPingImpl{
		config: config,
		stdout: stdout,
		pinger: pinger,
		logger: logger,
	}, nil
}

// Run does start of the application logic, returns an error if something goes wrong, nil otherwise
func (httpPingImpl *httpPingImpl) Run() error {

	config := httpPingImpl.config
	stdout := httpPingImpl.stdout

	ic := make(chan os.Signal, 1)

	signal.Notify(ic, os.Interrupt)

	ch := httpPingImpl.pinger.Ping()

	_, _ = fmt.Fprintf(stdout, "HTTP-PING %s %s\n\n", httpPingImpl.pinger.URL(), config.Method)

	successes := 0
	attempts := 0
	var latencies []stats.Measure

	var loop = true
	for loop {
		select {
		case measure := <-ch:
			if measure == nil {
				loop = false
			} else {
				httpPingImpl.logger.onMeasure(measure, attempts)
				attempts++
				if !measure.IsFailure {
					successes++
					latencies = append(latencies, measure.TotalTime)
					if config.AudibleBell {
						_, _ = fmt.Fprintf(stdout, "\a")
					}
				}
			}
		case <-ic:
			loop = false
		}
	}
	var lossRate = float64(0)
	if attempts > 0 {
		lossRate = float64(100*(attempts-successes)) / float64(attempts)
	}

	httpPingImpl.logger.onClose(int64(attempts), int64(successes), lossRate, stats.PingStatsFromLatencies(latencies))
	return nil
}

type logger interface {
	onMeasure(httpMeasure *HTTPMeasure, id int)
	onClose(attempts int64, success int64, lossRate float64, pingStats *stats.PingStats)
}

type quietLogger struct {
	config *Config
	stdout io.Writer
	pinger Pinger
}

func newQuietLogger(config *Config, stdout io.Writer, pinger Pinger) logger {
	return &quietLogger{config: config, stdout: stdout, pinger: pinger}
}

func (quietLogger *quietLogger) onStart() {}

func (quietLogger *quietLogger) onMeasure(_ *HTTPMeasure, _ int) {
}

func (quietLogger *quietLogger) onClose(attempts int64, successes int64, lossRate float64, pingStats *stats.PingStats) {

	_, _ = fmt.Fprintf(quietLogger.stdout, "--- %s ping statistics ---\n", quietLogger.pinger.URL())

	_, _ = fmt.Fprintf(quietLogger.stdout, "%d requests sent, %d answers received, %.1f%% loss\n", attempts, successes, lossRate)

	if successes > 0 {
		_, _ = fmt.Fprintf(quietLogger.stdout, "%s\n", pingStats.String())
	}
}

type standardLogger struct {
	config *Config
	stdout io.Writer
	pinger Pinger
}

func newStandardLogger(config *Config, stdout io.Writer, pinger Pinger) logger {
	return &standardLogger{config: config, stdout: stdout, pinger: pinger}
}

func (standardLogger *standardLogger) onMeasure(measure *HTTPMeasure, id int) {

	if measure.IsFailure {
		_, _ = fmt.Fprintf(standardLogger.stdout, "%4d: Error: %s\n", id, measure.FailureCause)
		return
	}
	_, _ = fmt.Fprintf(standardLogger.stdout, "%8d: %s, code=%d, size=%d bytes, time=%.1f ms\n", id, measure.RemoteAddr, measure.StatusCode, measure.Bytes, measure.TotalTime.ToFloat(time.Millisecond))

}

func (standardLogger *standardLogger) onClose(attempts int64, successes int64, lossRate float64, pingStats *stats.PingStats) {
	_, _ = fmt.Fprintf(standardLogger.stdout, "\n")
	_, _ = fmt.Fprintf(standardLogger.stdout, "--- %s ping statistics ---\n", standardLogger.pinger.URL())

	_, _ = fmt.Fprintf(standardLogger.stdout, "%d requests sent, %d answers received, %.1f%% loss\n", attempts, successes, lossRate)

	if successes > 0 {
		_, _ = fmt.Fprintf(standardLogger.stdout, "%s\n", pingStats.String())
	}
}

type verboseLogger struct {
	config     *Config
	stdout     io.Writer
	measureSum *HTTPMeasure
	pinger     Pinger
}

func newVerboseLogger(config *Config, stdout io.Writer, pinger Pinger) logger {
	return &verboseLogger{config: config, stdout: stdout, pinger: pinger,
		measureSum: &HTTPMeasure{
			DNSResolution: stats.MeasureNotValid,
			TCPHandshake:  stats.MeasureNotValid,
			TLSDuration:   stats.MeasureNotValid,
		},
	}
}

func (verboseLogger *verboseLogger) onMeasure(measure *HTTPMeasure, id int) {

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
