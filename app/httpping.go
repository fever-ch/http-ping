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

	_, _ = fmt.Fprintf(stdout, "HTTP-PING %s %s\n\n", httpPingImpl.pinger.URL(), config.Method)

	measuresChannel := httpPingImpl.pinger.Ping()

	successes := 0
	attempts := 0
	var latencies []stats.Measure
	ticker := time.NewTicker(5 * time.Second)
	tickerChan := make(<-chan time.Time)
	ticker.Stop()
	tpuStarted := false
	tputMeasurer := newTputMeasurer()

	loop := true
	for loop {
		select {

		case _ = <-tickerChan:

			m := tputMeasurer.Measure()
			httpPingImpl.logger.onTick(m)

		case measure := <-measuresChannel:
			if measure == nil {
				loop = false
			} else {
				httpPingImpl.logger.onMeasure(measure, attempts)
				attempts++
				if !measure.IsFailure {
					if config.Tput && !tpuStarted {
						tputMeasurer.Measure()
						tickerChan = (time.NewTicker(config.TputRefresh)).C
						tpuStarted = true
					}
					tputMeasurer.Count(measure.TotalTime)

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

func countToString(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}
