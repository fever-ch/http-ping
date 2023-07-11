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

func newWebClient(config *Config, runtimeConfig *RuntimeConfig) (*webClientImpl, error) {
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

	if webClient.config.DisableHTTP2 {
		tlsNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
	}

	webClient.httpClient = &http.Client{
		Timeout: webClient.config.Wait,
		Transport: &http.Transport{
			Proxy:       http.ProxyFromEnvironment,
			DialContext: dialCtx,

			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.NoCheckCertificate,
			},
			DisableCompression: config.DisableCompression,
			ForceAttemptHTTP2:  !webClient.config.DisableHTTP2,
			MaxIdleConns:       10,
			DisableKeepAlives:  config.DisableKeepAlive,
			IdleConnTimeout:    config.Interval + config.Wait,
			TLSNextProto:       tlsNextProto,
		},
	}

	return &webClient, nil
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

	totalTimer := newTimer()
	connTimer := newTimer()
	dnsTimer := newTimer()
	tlsTimer := newTimer()
	tcpTimer := newTimer()
	reqTimer := newTimer()
	waitTimer := newTimer()
	responseTimer := newTimer()

	clientTrace := &httptrace.ClientTrace{
		TLSHandshakeStart: func() {
			tlsTimer.start()
		},

		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			tlsTimer.stop()
		},
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsTimer.start()
		},

		DNSDone: func(info httptrace.DNSDoneInfo) {
			dnsTimer.stop()
		},

		GetConn: func(hostPort string) {
			connTimer.start()
		},

		GotConn: func(info httptrace.GotConnInfo) {
			remoteAddr = info.Conn.RemoteAddr().String()
			connTimer.stop()
			reqTimer.start()
			reused = info.Reused
		},

		WroteRequest: func(info httptrace.WroteRequestInfo) {
			reqTimer.stop()
			waitTimer.start()
		},

		GotFirstResponseByte: func() {
			waitTimer.stop()
			responseTimer.start()
		},
	}

	ctx := sockettrace.WithTrace(context.Background(),
		&sockettrace.ConnTrace{
			Read: func(i int) {
				atomic.AddInt64(&webClient.reads, int64(i))
			},
			Write: func(i int) {
				atomic.AddInt64(&webClient.writes, int64(i))
			},
			TCPStart: func() {
				tcpTimer.start()
			},
			TCPEstablished: func() {
				tcpTimer.stop()
			},
		})

	traceCtx := httptrace.WithClientTrace(ctx, clientTrace)

	req = req.WithContext(traceCtx)

	webClient.prepareReq(req)

	totalTimer.start()
	res, err := webClient.httpClient.Do(req)

	if err != nil {
		return &HTTPMeasure{
			IsFailure:    true,
			FailureCause: err.Error(),
		}
	}

	s, err := io.Copy(io.Discard, res.Body)
	if err != nil {
		return &HTTPMeasure{
			IsFailure:    true,
			FailureCause: "I/O error while reading payload",
		}
	}

	_ = res.Body.Close()
	responseTimer.stop()
	totalTimer.stop()

	if webClient.config.DisableKeepAlive {
		webClient.httpClient.CloseIdleConnections()
	}

	failed := false
	failureCause := ""

	if res.StatusCode/100 == 5 && !webClient.config.IgnoreServerErrors {
		failed = true
		failureCause = "Server-side error"
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

	return &HTTPMeasure{
		Proto:        res.Proto,
		TotalTime:    totalTimer.measure(),
		StatusCode:   res.StatusCode,
		Bytes:        s,
		InBytes:      i,
		OutBytes:     o,
		SocketReused: reused,
		Compressed:   !res.Uncompressed,
		TLSEnabled:   res.TLS != nil,
		TLSVersion:   tlsVersion,

		DNSResolution:     dnsTimer.measure(),
		TCPHandshake:      tcpTimer.measure(),
		TLSDuration:       tlsTimer.measure(),
		ConnEstablishment: connTimer.measure(),
		RequestSending:    reqTimer.measure(),
		Wait:              waitTimer.measure(),
		ResponseIngesting: responseTimer.measure(),

		RemoteAddr: remoteAddr,

		IsFailure:    failed,
		FailureCause: failureCause,
		Headers:      &res.Header,
	}

}
