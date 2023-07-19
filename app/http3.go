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
	"context"
	"crypto/tls"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"net/http"
	"regexp"
	"strings"
)

func newHttp3RoundTripper(config *Config, runtimeConfig *RuntimeConfig, w *webClientImpl) (http.RoundTripper, error) {
	return &http3.RoundTripper{

		DisableCompression: config.DisableCompression,
		Dial: func(ctx context.Context, addr string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlyConnection, error) {

			connAddr, e := w.resolver.resolveConn(addr)
			if e != nil {
				return nil, e
			}
			runtimeConfig.ResolvedConnAddress = connAddr
			dae, err := quic.DialAddrEarly(ctx, connAddr, tlsCfg, cfg)

			return wrapEarlyConnection(dae, w), err
		},

		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.NoCheckCertificate,
		},
		QuicConfig: &(quic.Config{}),
	}, nil
}

func CheckAltSvcH3Header(h http.Header) *string {
	for k, entries := range h {
		if strings.ToUpper(k) == "ALT-SVC" {
			for _, entry := range entries {
				if value := CheckAltSvcH3(entry); value != nil {
					return value
				}
			}
		}
	}
	return nil
}

var fieldRx = regexp.MustCompile(`^\s*([a-zA-Z0-9-]+)=(.*)$`)

func CheckAltSvcH3(s string) *string {
	for _, prop := range strings.Split(s, ";") {
		kv := fieldRx.FindStringSubmatch(prop)

		if kv[1] == "h3" {
			vl := kv[2]
			if len(vl) >= 2 && vl[0] == '"' && vl[len(vl)-1] == '"' {
				unquoted := vl[1 : len(vl)-1]
				return &unquoted
			}
			return &vl
		}
	}
	return nil
}
