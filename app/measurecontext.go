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
	"fever.ch/http-ping/net/sockettrace"
	"fever.ch/http-ping/stats"
	"net/http/httptrace"
)

type measureContext struct {
	timerRegistry *stats.TimerRegistry
	webClientImpl *webClientImpl
	remoteAddr    string
	reused        bool
}

func newMeasureContext(impl *webClientImpl) *measureContext {
	return &measureContext{
		timerRegistry: stats.NewTimersCollection(),
		webClientImpl: impl,
	}
}

func (measureContext *measureContext) getMeasures() *stats.MeasuresCollection {
	return measureContext.timerRegistry.Measure()
}

func (measureContext *measureContext) getClientTrace() *httptrace.ClientTrace {

	return &httptrace.ClientTrace{
		TLSHandshakeStart: func() {
			measureContext.timerRegistry.Get(stats.TLS).Start()
		},

		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			measureContext.timerRegistry.Get(stats.TLS).Stop()
		},
		DNSStart: func(info httptrace.DNSStartInfo) {
			measureContext.timerRegistry.Get(stats.DNS).Start()
		},

		DNSDone: func(info httptrace.DNSDoneInfo) {
			measureContext.timerRegistry.Get(stats.DNS).Stop()
		},

		GetConn: func(hostPort string) {
			measureContext.timerRegistry.Get(stats.Conn).Start()
		},

		GotConn: func(info httptrace.GotConnInfo) {
			measureContext.remoteAddr = info.Conn.RemoteAddr().String()
			measureContext.timerRegistry.Get(stats.Conn).Stop()
			measureContext.timerRegistry.Get(stats.Req).Start()
			measureContext.timerRegistry.Get(stats.ReqAndWait).StartForce()
			measureContext.reused = info.Reused
		},

		WroteRequest: func(info httptrace.WroteRequestInfo) {
			measureContext.timerRegistry.Get(stats.Req).Stop()
			measureContext.timerRegistry.Get(stats.Wait).Start()
		},

		GotFirstResponseByte: func() {
			measureContext.timerRegistry.Get(stats.Wait).Stop()
			measureContext.timerRegistry.Get(stats.ReqAndWait).Stop()

			measureContext.timerRegistry.Get(stats.Resp).Start()
		},
	}

}

func (measureContext *measureContext) getConnTrace() *sockettrace.ConnTrace {
	return &sockettrace.ConnTrace{
		Read: func(i int) {
			measureContext.webClientImpl.reads += int64(i)
		},
		Write: func(i int) {
			measureContext.webClientImpl.writes += int64(i)
		},
		TCPStart: func() {
			measureContext.timerRegistry.Get(stats.TCP).Start()
		},
		TCPEstablished: func() {
			measureContext.timerRegistry.Get(stats.TCP).Stop()
		},
	}
}

func (measureContext *measureContext) ctx() context.Context {
	return httptrace.WithClientTrace(
		sockettrace.WithTrace(
			context.Background(),
			measureContext.getConnTrace()),
		measureContext.getClientTrace())
}

func (measureContext *measureContext) start() {
	measureContext.timerRegistry.Get(stats.Total).Start()
	measureContext.timerRegistry.Get(stats.ReqAndWait).Start() // for HTTP/3 with keep-alive
}

func (measureContext *measureContext) startIngestion() {
	measureContext.timerRegistry.Get(stats.ReqAndWait).Stop()
	measureContext.timerRegistry.Get(stats.Resp).Start()
}

func (measureContext *measureContext) globalStop() {
	measureContext.timerRegistry.Get(stats.Resp).Stop()
	measureContext.timerRegistry.Get(stats.Total).Stop()
}
