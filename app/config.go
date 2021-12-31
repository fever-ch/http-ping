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
	"time"
)

type pair struct {
	Name  string
	Value string
}

// Cookie is a data structure which represents the basic info about a cookie (Name and Value)
type Cookie pair

// Header is a data structure which represents the basic info about a HTTP header (Name and Value)
type Header pair

// Parameter is a data structure which represents a request parameter (Name and Value)
type Parameter pair

// Config defines the multiple parameters which can be passed to NewHTTPPing
type Config struct {
	IPProtocol         string
	Interval           time.Duration
	Count              int64
	Target             string
	Method             string
	UserAgent          string
	Wait               time.Duration
	DisableKeepAlive   bool
	LogLevel           int8
	ConnTarget         string
	NoCheckCertificate bool
	Cookies            []Cookie
	Headers            []Header
	Parameters         []Parameter
	IgnoreServerErrors bool
	ExtraParam         bool
	DisableCompression bool
	AudibleBell        bool
	Referrer           string
	AuthUsername       string
	AuthPassword       string
	DisableHTTP2       bool
	FullDNS            bool
	DNSServer          string
	CacheDNSRequests   bool
	KeepCookies        bool
	FollowRedirects    bool
	Workers            int
	Tput               bool
}

// RuntimeConfig defines the parameters which can be passed to NewPinger and NewWebClient
type RuntimeConfig struct {
	RedirectCallBack func(url string)
}
