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
	"fmt"
	"net/http"
	"sync"
	"time"
)

// HTTPMeasure is the out of a measurement done as an HTTP ping
type HTTPMeasure struct {
	Proto string

	StatusCode   int
	Bytes        int64
	InBytes      int64
	OutBytes     int64
	SocketReused bool
	Compressed   bool
	RemoteAddr   string
	TLSEnabled   bool
	TLSVersion   string
	AltSvcH3     *string

	MeasuresCollection *stats.MeasuresCollection

	IsFailure    bool
	FailureCause string
	Headers      *http.Header
}

// Pinger does the calls to the actual HTTP/S component
type Pinger interface {
	Ping() <-chan *HTTPMeasure

	URL() string
}

type pingerImpl struct {
	clientBuilder WebClientBuilder
	config        *Config
}

// NewPinger builds a new pingerImpl
func NewPinger(config *Config, runtimeConfig *RuntimeConfig, logger ConsoleLogger) (Pinger, error) {

	pinger := pingerImpl{}

	pinger.config = config

	client, err := NewWebClientBuilder(config, runtimeConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("%s (%s)", err, config.IPProtocol)
	}

	pinger.clientBuilder = client

	return &pinger, nil
}

func (pinger *pingerImpl) URL() string {
	return pinger.clientBuilder.URL()
}

// Ping actually does the pinging specified in config
func (pinger *pingerImpl) Ping() <-chan *HTTPMeasure {
	measures := make(chan *HTTPMeasure)

	var wg sync.WaitGroup

	if pinger.config.FollowRedirects {
		i := pinger.clientBuilder.NewInstance()
		i.DoMeasure(true)
		pinger.clientBuilder.SetURL(i.GetURL())
	}

	for i := 0; i < pinger.config.Workers; i++ {
		wg.Add(1)

		go func() {

			client := pinger.clientBuilder.NewInstance()

			defer wg.Done()

			if !pinger.config.DisableKeepAlive {
				client.DoMeasure(pinger.config.FollowRedirects)
				time.Sleep(time.Second)
			}

			for a := int64(0); a < pinger.config.Count; a++ {
				measures <- client.DoMeasure(false)

				if a < pinger.config.Count-1 {
					time.Sleep(pinger.config.Interval)
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(measures)
	}()
	return measures
}
