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
	"fmt"
	"github.com/fever-ch/http-ping/stats"
	"io"
	"os"
	"os/signal"
	"time"
)

// HTTPPing actually does the pinging specified in config
func HTTPPing(config *Config, stdout io.Writer) {

	ic := make(chan os.Signal, 1)

	signal.Notify(ic, os.Interrupt)

	pinger, err := NewPinger(config)

	pinger.RedirectCallBack = func(url string) {
		_, _ = fmt.Fprintf(stdout, "   ─→     Redirected to %s\n\n", url)
	}

	if err != nil {
		_, _ = fmt.Fprintf(stdout, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	ch := pinger.Ping()

	_, _ = fmt.Fprintf(stdout, "HTTP-PING %s %s\n\n", pinger.client.url.String(), config.Method)

	var latencies []stats.Measure
	attempts, failures := 0, 0

	measureSum := &HTTPMeasure{
		DNSResolution: stats.MeasureNotValid,
		TCPHandshake:  stats.MeasureNotValid,
		TLSDuration:   stats.MeasureNotValid}

	var loop = true
	for loop {
		select {
		case measure := <-ch:
			if measure == nil {
				loop = false
			} else {
				if !measure.IsFailure {
					if config.LogLevel >= 1 {
						_, _ = fmt.Fprintf(stdout, "%8d: %s, code=%d, size=%d bytes, time=%.1f ms\n", attempts, measure.RemoteAddr, measure.StatusCode, measure.Bytes, measure.TotalTime.ToFloat(time.Millisecond))
					}
					if config.LogLevel == 2 {
						_, _ = fmt.Fprintf(stdout, "          proto=%s, socket reused=%t, compressed=%t\n", measure.Proto, measure.SocketReused, measure.Compressed)
						_, _ = fmt.Fprintf(stdout, "          network i/o: bytes read=%d, bytes written=%d\n", measure.InBytes, measure.OutBytes)

						if measure.TLSEnabled {
							_, _ = fmt.Fprintf(stdout, "          tls version=%s\n", measure.TLSVersion)
						}

						measureSum.TotalTime += measure.TotalTime

						measureSum.ConnEstablishment = measureSum.ConnEstablishment.SumIfValid(measure.ConnEstablishment)
						measureSum.DNSResolution = measureSum.DNSResolution.SumIfValid(measure.DNSResolution)
						measureSum.TCPHandshake = measureSum.TCPHandshake.SumIfValid(measure.TCPHandshake)
						measureSum.TLSDuration = measureSum.TLSDuration.SumIfValid(measure.TLSDuration)
						measureSum.RequestSending += measure.RequestSending
						measureSum.Wait += measure.Wait
						measureSum.ResponseIngesting += measure.ResponseIngesting

						_, _ = fmt.Fprintf(stdout, "\n")

						_, _ = fmt.Fprintf(stdout, "          latency contributions:\n")

						drawMeasure(measure, stdout)

						_, _ = fmt.Fprintf(stdout, "\n")
					}
					latencies = append(latencies, measure.TotalTime)

					if config.AudibleBell {
						_, _ = fmt.Fprintf(stdout, "\a")
					}
				} else {
					if config.LogLevel >= 1 {
						_, _ = fmt.Fprintf(stdout, "%4d: Error: %s\n", attempts, measure.FailureCause)
					}
					failures++
				}
				attempts++
			}
		case <-ic:
			loop = false
		}
	}

	if config.LogLevel != 2 {
		_, _ = fmt.Fprintf(stdout, "\n")
	}
	fmt.Printf("--- %s ping statistics ---\n", pinger.client.url.String())
	var lossRate = float64(0)
	if attempts > 0 {
		lossRate = float64(100*failures) / float64(attempts)
	}

	success := int64(attempts - failures)

	_, _ = fmt.Fprintf(stdout, "%d requests sent, %d answers received, %.1f%% loss\n", attempts, attempts-failures, lossRate)

	if len(latencies) > 0 {
		_, _ = fmt.Fprintf(stdout, "%s\n", stats.PingStatsFromLatencies(latencies).String())

		if config.LogLevel == 2 {
			measureSum.TotalTime = measureSum.TotalTime.Divide(success)
			measureSum.ConnEstablishment = measureSum.ConnEstablishment.Divide(success)
			measureSum.DNSResolution = measureSum.DNSResolution.Divide(success)
			measureSum.TCPHandshake = measureSum.TCPHandshake.Divide(success)
			measureSum.TLSDuration = measureSum.TLSDuration.Divide(success)
			measureSum.RequestSending = measureSum.RequestSending.Divide(success)
			measureSum.Wait = measureSum.Wait.Divide(success)
			measureSum.ResponseIngesting = measureSum.ResponseIngesting.Divide(success)

			measureSum.TLSEnabled = measureSum.TLSDuration > 0

			_, _ = fmt.Fprintf(stdout, "\naverage latency contributions:\n")

			drawMeasure(measureSum, stdout)
		}
	}

}

func drawMeasure(measure *HTTPMeasure, stdout io.Writer) {
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

	l := makeTreeList(&entries)

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

func makeTreeList(root *measureEntry) []measureEntryVisit {
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
