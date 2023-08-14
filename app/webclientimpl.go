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
	"fever.ch/http-ping/net/sockettrace"
	"fever.ch/http-ping/stats"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptrace"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

type webClientImpl struct {
	httpClient    *http.Client
	connTarget    string
	config        *Config
	runtimeConfig *RuntimeConfig
	url           *url.URL
	resolver      *resolver

	writes int64
	reads  int64
}

func (webClient *webClientImpl) updateConnTarget() {
	if webClient.config.ConnTarget == "" {
		webClient.resolver = newResolver(webClient.config)

		webClient.connTarget = webClient.url.Hostname()
		ipAddr := webClient.url.Hostname()

		var port = webClient.url.Port()
		if port == "" {
			port = portMap[webClient.url.Scheme]
		}

		if strings.Contains(ipAddr, ":") {
			webClient.connTarget = fmt.Sprintf("[%s]:%s", ipAddr, port)
		} else {
			webClient.connTarget = fmt.Sprintf("%s:%s", ipAddr, port)
		}
	} else {
		webClient.connTarget = webClient.config.ConnTarget
	}
}

func newHttp2RoundTripper(config *Config, runtimeConfig *RuntimeConfig, w *webClientImpl) (http.RoundTripper, error) {

	webClient := webClientImpl{config: config, runtimeConfig: runtimeConfig}
	parsedURL, err := url.Parse(config.Target)
	if err != nil {
		return nil, err
	}
	webClient.url = parsedURL

	webClient.updateConnTarget()

	dialer := &net.Dialer{}

	startDNSHook := func(ctx context.Context) {
		trace := httptrace.ContextClientTrace(ctx)
		if trace != nil && trace.DNSStart != nil {
			trace.DNSStart(httptrace.DNSStartInfo{})
		}
	}

	stopDNSHook := func(ctx context.Context) {
		trace := httptrace.ContextClientTrace(ctx)
		if trace != nil && trace.DNSDone != nil {
			trace.DNSDone(httptrace.DNSDoneInfo{})
		}
	}

	dialCtx := func(ctx context.Context, network, addr string) (net.Conn, error) {
		var ipaddr string

		startDNSHook(ctx)

		if webClient.config.ConnTarget == "" {
			resolvedIpaddr, err := webClient.resolver.resolveConn(webClient.connTarget)

			if err != nil {
				return nil, err
			}
			ipaddr = resolvedIpaddr
		} else {
			ipaddr = webClient.config.ConnTarget
		}
		stopDNSHook(ctx)

		return sockettrace.NewSocketTrace(ctx, dialer, network, ipaddr)
	}

	var tlsNextProto map[string]func(string, *tls.Conn) http.RoundTripper = nil

	if webClient.config.Http1 {
		tlsNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
	}

	return &http.Transport{
		Proxy:       http.ProxyFromEnvironment,
		DialContext: dialCtx,

		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.NoCheckCertificate,
		},
		DisableCompression: config.DisableCompression,
		ForceAttemptHTTP2:  !webClient.config.Http1,
		MaxIdleConns:       10,
		DisableKeepAlives:  config.DisableKeepAlive,
		IdleConnTimeout:    config.Interval + config.Wait,
		TLSNextProto:       tlsNextProto,
	}, nil
}

func newTransport(config *Config, runtimeConfig *RuntimeConfig, w *webClientImpl) (http.RoundTripper, error) {

	if config.Http3 {
		return newHttp3RoundTripper(config, runtimeConfig, w)
	}
	return newHttp2RoundTripper(config, runtimeConfig, w)
}

func newWebClient(config *Config, runtimeConfig *RuntimeConfig) (*webClientImpl, error) {
	webClient := &webClientImpl{}

	err := webClient.update(config, runtimeConfig)

	if err != nil {
		return nil, err
	}

	return webClient, nil
}

func (webClient *webClientImpl) update(config *Config, runtimeConfig *RuntimeConfig) error {
	webClient.config = config
	webClient.runtimeConfig = runtimeConfig
	parsedURL, err := url.Parse(config.Target)
	if err != nil {
		return err
	}
	webClient.url = parsedURL

	webClient.updateConnTarget()

	tr, _ := newTransport(config, runtimeConfig, webClient)

	webClient.httpClient = &http.Client{
		Timeout:   webClient.config.Wait,
		Transport: tr,
	}

	return nil
}

func (webClient *webClientImpl) URL() string {
	return webClient.url.String()
}

func (webClient *webClientImpl) SetURL(url *url.URL) {
	webClient.url = url
}

func (webClient *webClientImpl) GetURL() *url.URL {
	return webClient.url
}

func (webClient *webClientImpl) checkRedirectFollow(req *http.Request, _ []*http.Request) error {
	webClient.config.Target = req.URL.String()
	webClient.url = req.URL
	if webClient.runtimeConfig.RedirectCallBack != nil {
		webClient.runtimeConfig.RedirectCallBack(req.URL.String())
	}
	webClient.updateConnTarget()
	return nil
}

func (webClient *webClientImpl) prepareReq(req *http.Request) {
	if len(webClient.config.Parameters) > 0 || webClient.config.ExtraParam {
		q := req.URL.Query()

		if webClient.config.ExtraParam {
			q.Add("extra_parameter_http_ping", fmt.Sprintf("%d", time.Now().UnixMicro()))
		}

		for _, c := range webClient.config.Parameters {
			q.Add(c.Name, c.Value)
		}
		req.URL.RawQuery = q.Encode()
	}

	req.Header.Set("User-Agent", webClient.config.UserAgent)
	if webClient.config.Referrer != "" {
		req.Header.Set("Referer", webClient.config.Referrer)
	}

	if webClient.config.AuthUsername != "" || webClient.config.AuthPassword != "" {
		req.SetBasicAuth(webClient.config.AuthUsername, webClient.config.AuthPassword)
	}

	// Host is considered as a special header in net/http, for simplicity we use here a common way to handle both
	for _, header := range webClient.config.Headers {
		if strings.ToLower(header.Name) != "host" {
			req.Header.Set(header.Name, header.Value)
		} else {
			req.Host = header.Value
		}
	}

}

// DoMeasure evaluates the latency to a specific HTTP/S server
func (webClient *webClientImpl) DoMeasure(followRedirect bool) *HTTPMeasure {

	if followRedirect {
		webClient.httpClient.CheckRedirect = webClient.checkRedirectFollow
	} else {
		webClient.httpClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	req, _ := http.NewRequest(webClient.config.Method, webClient.config.Target, nil)

	if webClient.httpClient.Jar == nil || !webClient.config.KeepCookies {
		jar, _ := cookiejar.New(nil)
		var cookies []*http.Cookie
		for _, c := range webClient.config.Cookies {
			cookies = append(cookies, &http.Cookie{Name: c.Name, Value: c.Value})
		}

		jar.SetCookies(webClient.url, cookies)
		webClient.httpClient.Jar = jar
	}

	var reused bool
	var remoteAddr string

	timerRegistry := stats.NewTimersCollection()

	clientTrace := &httptrace.ClientTrace{
		TLSHandshakeStart: func() {
			timerRegistry.Get(stats.TLS).Start()
		},

		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			timerRegistry.Get(stats.TLS).Stop()
		},
		DNSStart: func(info httptrace.DNSStartInfo) {
			timerRegistry.Get(stats.DNS).Start()
		},

		DNSDone: func(info httptrace.DNSDoneInfo) {
			timerRegistry.Get(stats.DNS).Stop()
		},

		GetConn: func(hostPort string) {
			timerRegistry.Get(stats.Conn).Start()
		},

		GotConn: func(info httptrace.GotConnInfo) {
			remoteAddr = info.Conn.RemoteAddr().String()
			timerRegistry.Get(stats.Conn).Stop()
			timerRegistry.Get(stats.Req).Start()
			timerRegistry.Get(stats.ReqAndWait).Start()
			reused = info.Reused
		},

		WroteRequest: func(info httptrace.WroteRequestInfo) {
			timerRegistry.Get(stats.Req).Stop()
			timerRegistry.Get(stats.Wait).Start()
		},

		GotFirstResponseByte: func() {
			timerRegistry.Get(stats.Wait).Stop()
			timerRegistry.Get(stats.ReqAndWait).Stop()

			timerRegistry.Get(stats.Resp).Start()
		},
	}

	connTrace := &sockettrace.ConnTrace{
		Read: func(i int) {
			atomic.AddInt64(&webClient.reads, int64(i))
		},
		Write: func(i int) {
			atomic.AddInt64(&webClient.writes, int64(i))
		},
		TCPStart: func() {
			timerRegistry.Get(stats.TCP).Start()
		},
		TCPEstablished: func() {
			timerRegistry.Get(stats.TCP).Stop()
		},
	}

	traceCtx :=
		httptrace.WithClientTrace(
			sockettrace.WithTrace(
				context.Background(),
				connTrace),
			clientTrace)

	req = req.WithContext(traceCtx)

	webClient.prepareReq(req)

	timerRegistry.Get(stats.Total).Start()

	res, err := webClient.httpClient.Do(req)

	if err != nil {
		return &HTTPMeasure{
			IsFailure:    true,
			FailureCause: err.Error(),
		}
	}

	altSvcH3 := ""
	if res != nil {
		if val := CheckAltSvcH3Header(res.Header); val != nil {
			altSvcH3 = *val
		}
	}

	if altSvcH3 != "" && !strings.HasPrefix(res.Proto, "HTTP/3") && !webClient.config.Http1 && !webClient.config.Http2 {
		_, _ = fmt.Printf("   ─→     server advertised HTTP/3 endpoint, using HTTP/3\n")

		if altSvcH3 == ":443" {
			// nothing
		} else if strings.HasPrefix(altSvcH3, ":") {
			webClient.url.Host = webClient.url.Host + altSvcH3
		} else {
			webClient.url.Host = altSvcH3
		}

		c := *webClient.config
		c.Http3 = true
		err := webClient.update(&c, webClient.runtimeConfig)

		if err != nil {
			return &HTTPMeasure{
				IsFailure:    true,
				FailureCause: err.Error(),
			}
		}
		return webClient.DoMeasure(followRedirect)
	}

	timerRegistry.Get(stats.ReqAndWait).Stop()
	timerRegistry.Get(stats.Resp).Start()

	s, err := io.Copy(io.Discard, res.Body)
	if err != nil {
		return &HTTPMeasure{
			IsFailure:    true,
			FailureCause: "I/O error while reading payload",
		}
	}

	_ = res.Body.Close()

	timerRegistry.Get(stats.Resp).Stop()
	timerRegistry.Get(stats.Total).Stop()

	if webClient.config.DisableKeepAlive {
		webClient.httpClient.CloseIdleConnections()
	}

	failed := false
	failureCause := ""

	if res.StatusCode/100 == 5 && !webClient.config.IgnoreServerErrors {
		failed = true
		failureCause = "Server-side error"
	}

	if strings.HasPrefix(res.Proto, "HTTP/1.") && webClient.config.Http2 {
		failed = true
		failureCause = "HTTP/2 not supported by server"
	}

	i := atomic.SwapInt64(&webClient.reads, 0)
	o := atomic.SwapInt64(&webClient.writes, 0)

	var tlsVersion string
	if res.TLS != nil {
		code := int(res.TLS.Version) - 0x0301
		if code >= 0 {
			tlsVersion = fmt.Sprintf("TLS-1.%d", code)
		} else {
			tlsVersion = "SSL-3"
		}
	}

	if remoteAddr == "" {
		remoteAddr = webClient.runtimeConfig.ResolvedConnAddress
	}

	return &HTTPMeasure{
		Proto: res.Proto,
		//TotalTime:    timerRegistry.get(Total).measure(),
		StatusCode:   res.StatusCode,
		Bytes:        s,
		InBytes:      i,
		OutBytes:     o,
		SocketReused: reused,
		Compressed:   !res.Uncompressed,
		TLSEnabled:   res.TLS != nil,
		TLSVersion:   tlsVersion,
		AltSvcH3:     altSvcH3,

		MeasuresCollection: timerRegistry.Measure(),

		RemoteAddr: remoteAddr,

		IsFailure:    failed,
		FailureCause: failureCause,
		Headers:      &res.Header,
	}

}
