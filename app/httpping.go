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

	ticker := time.NewTicker(5 * time.Second)
	tickerChan := make(<-chan time.Time)
	ticker.Stop()
	tpuStarted := false
	throughputMeasurer := newThroughputMeasurer()

	loop := true
	for loop {
		select {

		case <-tickerChan:
			m := throughputMeasurer.Measure()
			httpPingImpl.logger.onTick(m)

		case measure := <-measuresChannel:
			if measure == nil {
				loop = false
			} else {
				httpPingImpl.logger.onMeasure(measure)
				if !measure.IsFailure {
					if config.Throughput && !tpuStarted {
						throughputMeasurer.Measure()
						tickerChan = (time.NewTicker(config.ThroughputRefresh)).C
						tpuStarted = true
					}
					throughputMeasurer.Count(measure.TotalTime)

					if config.AudibleBell {
						_, _ = fmt.Fprintf(stdout, "\a")
					}
				}
			}
		case <-ic:
			loop = false
		}
	}

	httpPingImpl.logger.onClose()
	if httpPingImpl.config.Throughput {
		httpPingImpl.logger.onThroughputClose()
	}
	return nil
}
