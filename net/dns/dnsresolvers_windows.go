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

//go:build windows

package dns

import (
	"log"

	wmi "github.com/yusufpapurcu/wmi"
)

type Win32_NetworkAdapterConfiguration struct {
	DNSServerSearchOrder []string
}

func GetDNSServers() ([]string, error) {
	var s []Win32_NetworkAdapterConfiguration
	err := wmi.Query("SELECT * FROM Win32_NetworkAdapterConfiguration WHERE IPEnabled = TRUE", &s)
	if err != nil {
		log.Fatal(err)
	}

	var list []string
	for _, item := range s {
		for _, server := range item.DNSServerSearchOrder {

			list = append(list, server)
		}
	}
	return list, nil
}
