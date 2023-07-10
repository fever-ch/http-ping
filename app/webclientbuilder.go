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
	"net/url"
	"strings"
)

// WebClientBuilder represents an HTTP/S clientBuilder designed to do performance analysis
type WebClientBuilder interface {
	URL() string

	SetURL(url *url.URL)

	NewInstance() WebClient
}

type webClientBuilderImpl struct {
	connTarget    string
	config        *Config
	runtimeConfig *RuntimeConfig
	url           *url.URL
	resolver      *resolver
}

// NewWebClientBuilder builds a new instance of webClientImpl which will provides functions for Http-Ping
func NewWebClientBuilder(config *Config, runtimeConfig *RuntimeConfig) (WebClientBuilder, error) {
	webClient := webClientBuilderImpl{config: config, runtimeConfig: runtimeConfig}
	parsedURL, err := url.Parse(config.Target)
	if err != nil {
		return nil, err
	}
	webClient.url = parsedURL

	webClient.updateConnTarget()

	return &webClient, nil
}

func (webClientBuilder *webClientBuilderImpl) URL() string {
	return webClientBuilder.url.String()
}

func (webClientBuilder *webClientBuilderImpl) SetURL(url *url.URL) {
	webClientBuilder.url = url
}

func (webClientBuilder *webClientBuilderImpl) GetURL() *url.URL {
	return webClientBuilder.url
}

func (webClientBuilder *webClientBuilderImpl) NewInstance() WebClient {
	w, _ := newWebClient(webClientBuilder.config, webClientBuilder.runtimeConfig)

	w.url = webClientBuilder.url
	w.updateConnTarget()

	return w
}

func (webClientBuilder *webClientBuilderImpl) updateConnTarget() {
	if webClientBuilder.config.ConnTarget == "" {
		webClientBuilder.resolver = newResolver(webClientBuilder.config)

		webClientBuilder.connTarget = webClientBuilder.url.Hostname()
		ipAddr := webClientBuilder.url.Hostname()

		var port = webClientBuilder.url.Port()
		if port == "" {
			port = portMap[webClientBuilder.url.Scheme]
		}

		if strings.Contains(ipAddr, ":") {
			webClientBuilder.connTarget = fmt.Sprintf("[%s]:%s", ipAddr, port)
		} else {
			webClientBuilder.connTarget = fmt.Sprintf("%s:%s", ipAddr, port)
		}
	} else {
		webClientBuilder.connTarget = webClientBuilder.config.ConnTarget
	}
}
