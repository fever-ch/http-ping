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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithEmbeddedWebServer(t *testing.T) {

	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/500":
				http.Error(w, "missing key", http.StatusInternalServerError)
			case "/404":
				http.Error(w, "missing key", http.StatusNotFound)
			default:
				_, _ = w.Write([]byte("Hello"))
			}
		}))

	url := ts.URL

	var webClient WebClientBuilder
	var measure *HTTPMeasure

	webClient, _ = NewWebClientBuilder(&Config{Target: fmt.Sprintf("%s/500", url), NoCheckCertificate: true}, &RuntimeConfig{}, nil)
	measure = webClient.NewInstance().DoMeasure(false)

	if !measure.IsFailure || measure.StatusCode != 500 {
		t.Errorf("Request to server should have failed, 500")
	}

	webClient, _ = NewWebClientBuilder(&Config{Target: fmt.Sprintf("%s/200", url), NoCheckCertificate: true}, &RuntimeConfig{}, nil)
	measure = webClient.NewInstance().DoMeasure(false)

	if measure.IsFailure || measure.StatusCode != 200 {
		t.Errorf("Request to server should have succeed, 200")
	}

	ts.Close()

	measure = webClient.NewInstance().DoMeasure(false)

	if !measure.IsFailure {
		t.Errorf("Request to server should have failed")
	}

}
