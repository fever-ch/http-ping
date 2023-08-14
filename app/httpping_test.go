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
	"bytes"
	"fever.ch/http-ping/stats"
	"io"
	"strings"
	"testing"
)

type PingerMock struct{}

func TestHTTPPing(t *testing.T) {
	b := bytes.NewBufferString("")
	instance, _ := NewHTTPPing(&Config{Count: 10}, b)
	instance.(*httpPingImpl).pinger = &PingerMock{}
	_ = instance.Run()

	out, _ := io.ReadAll(b)

	if !strings.Contains(string(out), "10 requests sent, 10 answers received, 0.0% loss") {
		t.Fatal("Result didn't match expectations")
	}
}

func (pingerMock *PingerMock) URL() string {
	return "https://www.google.com"
}

func (pingerMock *PingerMock) Ping() <-chan *HTTPMeasure {
	measures := make(chan *HTTPMeasure)

	go func() {
		defer close(measures)
		for i := 0; i < 10; i++ {
			measures <- &HTTPMeasure{
				MeasuresCollection: stats.NewMeasureRegistry(),
			}
		}
	}()

	return measures
}
