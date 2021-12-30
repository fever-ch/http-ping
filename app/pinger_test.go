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
	"testing"
)

type webClientMock struct{}

func TestPinger(t *testing.T) {
	wanted := 123
	pinger, _ := NewPinger(&Config{Workers: 1, Count: int64(wanted)}, &RuntimeConfig{})
	pinger.(*pingerImpl).client = &webClientMock{}
	ch := pinger.Ping()

	count := 0
	for range ch {
		count++
	}
	if count != 123 {
		t.Fatalf("%d != %d, number of measures didn't match", count, wanted)
	}
}

func (webClientMock *webClientMock) DoMeasure(_ bool) *HTTPMeasure {
	return &HTTPMeasure{}
}

func (webClientMock *webClientMock) URL() string {
	return "https://www.google.com"
}

func (webClientMock *webClientMock) Clone() WebClient {
	return webClientMock
}
