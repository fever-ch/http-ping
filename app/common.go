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

// Package app provides the main logic for the app
package app

import (
	"crypto/x509"
)

func init() {
	// load system cert pool once at the beginning to not impact further measures
	_, _ = x509.SystemCertPool()
}

var portMap = map[string]string{
	"http":  "80",
	"https": "443",
}
