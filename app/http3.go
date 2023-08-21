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
	"context"
	"crypto/tls"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"net"
	"net/http"
	"net/http/httptrace"
	"regexp"
	"strings"
)

func newHTTP3RoundTripper(config *Config, runtimeConfig *RuntimeConfig, w *webClientImpl) (http.RoundTripper, error) {
	if config.Method == http.MethodGet {
		config.Method = http3.MethodGet0RTT
	}
	return &http3.RoundTripper{
		DisableCompression: config.DisableCompression,
		Dial: func(ctx context.Context, addr string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlyConnection, error) {

			trace := httptrace.ContextClientTrace(ctx)

			traceGetConn(trace, addr)

			traceDNSStart(trace, addr)

			connAddr, e := w.resolver.resolveConn(addr)

			if e != nil {
				return nil, e
			}
			runtimeConfig.ResolvedConnAddress = connAddr

			traceDNSDone(trace, []net.IPAddr{})

			traceTLSHandshakeStart(trace)

			dae, err := quic.DialAddrEarly(ctx, connAddr, tlsCfg, cfg)
			if err != nil {
				return nil, err
			}

			traceTLSHandshakeDone(trace, tls.ConnectionState{})

			traceGotConn(trace, httptrace.GotConnInfo{Conn: connAdapter{remoteAddr: dae.RemoteAddr()}})

			return wrapEarlyConnection(dae, w), err
		},

		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.NoCheckCertificate,
		},
		QuicConfig: &(quic.Config{}),
	}, nil
}

func traceGotConn(trace *httptrace.ClientTrace, info httptrace.GotConnInfo) {
	if trace != nil && trace.GotConn != nil {
		trace.GotConn(info)
	}
}

func traceTLSHandshakeDone(trace *httptrace.ClientTrace, state tls.ConnectionState) {
	if trace != nil && trace.TLSHandshakeDone != nil {
		trace.TLSHandshakeDone(state, nil)
	}
}

func traceTLSHandshakeStart(trace *httptrace.ClientTrace) {
	if trace != nil && trace.TLSHandshakeStart != nil {
		trace.TLSHandshakeStart()
	}
}

func traceDNSDone(trace *httptrace.ClientTrace, addrs []net.IPAddr) {
	if trace != nil && trace.DNSDone != nil {
		trace.DNSDone(httptrace.DNSDoneInfo{
			Addrs: addrs,
		})
	}
}

func traceDNSStart(trace *httptrace.ClientTrace, addr string) {
	if trace != nil && trace.DNSStart != nil {
		trace.DNSStart(httptrace.DNSStartInfo{
			Host: addr,
		})
	}
}

func traceGetConn(trace *httptrace.ClientTrace, addr string) {
	if trace != nil && trace.GetConn != nil {
		trace.GetConn(addr)
	}
}

type connAdapter struct {
	*net.UDPConn
	remoteAddr net.Addr
}

func (c connAdapter) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func checkAltSvcH3Header(h http.Header) *string {
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
