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

type httpPingTestVersion struct {
	baseConfig      *Config
	advertisedHttp3 bool
	logger          *standardLogger
}

func (h *httpPingTestVersion) Run() error {
	http1 := h.checkHttp(func(c *Config) {
		c.Http1 = true
	})

	http2 := h.checkHttp(func(c *Config) {
		c.Http2 = true
	})

	http3 := h.checkHttp(func(c *Config) {
		c.Http3 = true
	})

	_, _ = h.logger.Printf("Checking available versions of HTTP protocol on " + h.baseConfig.Target)
	_, _ = h.logger.Printf("\n")
	_, _ = h.logger.Printf(" - v1  " + <-http1 + "\n")
	_, _ = h.logger.Printf(" - v2  " + <-http2 + "\n")
	_, _ = h.logger.Printf(" - v3  " + <-http3 + "\n")
	_, _ = h.logger.Printf("\n")
	if h.advertisedHttp3 {
		_, _ = h.logger.Printf("   (*) advertises HTTP/3 availability in HTTP headers\n")
	}
	return nil
}

func (h *httpPingTestVersion) checkHttp(prep func(*Config)) <-chan string {
	r := make(chan string)

	go func() {
		configCopy := *h.baseConfig

		configCopy.Http3 = false
		prep(&configCopy)

		rc := RuntimeConfig{}
		wc, _ := newWebClient(&configCopy, &rc)
		m := wc.DoMeasure(false)

		http3Advertisement := ""

		if m != nil && !m.IsFailure {
			if m.AltSvcH3 != nil {
				h.advertisedHttp3 = true
				http3Advertisement = " (*)"
			}

			r <- "\u001B[32m✓\u001B[0m " + m.Proto + http3Advertisement
		}
		r <- "\u001B[31m✗\u001B[0m not available"
	}()

	return r
}

// NewHTTPPing builds a new instance of HTTPPing or error if something goes wrong
func NewHTTPPing(config *Config, stdout io.Writer) (HTTPPing, error) {

	runtimeConfig := &RuntimeConfig{
		RedirectCallBack: func(url string) {
			_, _ = fmt.Fprintf(stdout, "   ─→     redirected to %s\n", url)
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

	if config.TestVersion {
		return &httpPingTestVersion{
			baseConfig: config,
			logger:     newStandardLogger(config, stdout, pinger),
		}, nil
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

	ic := make(chan os.Signal, 1)

	signal.Notify(ic, os.Interrupt)

	_, _ = httpPingImpl.logger.Printf("HTTP-PING %s %s\n\n", httpPingImpl.pinger.URL(), config.Method)

	measuresChannel := httpPingImpl.pinger.Ping()

	tickerChan := make(<-chan time.Time)
	tpuStarted := false
	throughputMeasurer := newThroughputMeasurer()

	loop := true
	first := true

	for loop {
		select {
		case <-tickerChan:
			m := throughputMeasurer.Measure()
			httpPingImpl.logger.onTick(m)

		case measure := <-measuresChannel:
			if measure == nil {
				loop = false
			} else {
				if first {
					_, _ = httpPingImpl.logger.Printf("\n")

					first = false
				}

				httpPingImpl.logger.onMeasure(measure)
				if config.Throughput && !tpuStarted {
					throughputMeasurer.Measure()
					tickerChan = (time.NewTicker(config.ThroughputRefresh)).C
					tpuStarted = true
				}
				if !measure.IsFailure {
					throughputMeasurer.Count(measure.MeasuresCollection.Get(stats.Total))

					httpPingImpl.logger.bell()
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
