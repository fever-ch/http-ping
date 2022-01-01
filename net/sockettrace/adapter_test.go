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

package sockettrace

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"testing"
)

func TestInterceptionOfHTTPRequest(t *testing.T) {

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				dialer := &net.Dialer{}
				return NewSocketTrace(ctx, dialer, network, addr)
			},
		},
	}

	ts := httptest.NewServer(nil)
	defer ts.Close()

	tcpStartedA := false
	tcpEstablishedA := false

	tcpStartedB := false
	tcpEstablishedB := false

	ctxA := WithTrace(context.Background(),
		&ConnTrace{
			Read: func(i int) {
			},
			Write: func(i int) {
			},
			TCPStart: func() {
				tcpStartedA = true
			},
			TCPEstablished: func() {
				tcpEstablishedA = true
			},
		})

	ctxB := WithTrace(ctxA,
		&ConnTrace{
			Read: func(i int) {
			},
			Write: func(i int) {
			},
			TCPStart: func() {
				tcpStartedB = true
			},
			TCPEstablished: func() {
				tcpEstablishedB = true
			},
		})

	clientTrace := &httptrace.ClientTrace{}

	traceCtx := httptrace.WithClientTrace(ctxB, clientTrace)

	req, _ := http.NewRequest("GET", ts.URL, nil)

	req = req.WithContext(traceCtx)

	_, err := httpClient.Do(req)

	if err != nil || !tcpStartedA || !tcpEstablishedA || !tcpStartedB || !tcpEstablishedB {
		t.Fatal("interception of TCP connection failed")
	}

}
