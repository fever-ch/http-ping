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
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"net"
	"net/http"
	"net/http/httptrace"
	"reflect"
	"regexp"
	"strings"
)

type http3EventKey struct{}

type Http3ClientTrace struct {
	GetConn func(hostPort string)

	// GotConn is called after a successful connection is
	// obtained. There is no hook for failure to obtain a
	// connection; instead, use the error from
	// Transport.RoundTrip.
	GotConn func()

	// DNSStart is called when a DNS lookup begins.
	DNSStart func(info httptrace.DNSStartInfo)

	// DNSDone is called when a DNS lookup ends.
	DNSDone func(info httptrace.DNSDoneInfo)

	QUICStart func()

	QUICDone func()
}

// compose modifies t such that it respects the previously-registered hooks in old,
// subject to the composition policy requested in t.Compose.
func (t *Http3ClientTrace) compose(old *Http3ClientTrace) {
	if old == nil {
		return
	}
	tv := reflect.ValueOf(t).Elem()
	ov := reflect.ValueOf(old).Elem()
	structType := tv.Type()
	for i := 0; i < structType.NumField(); i++ {
		tf := tv.Field(i)
		hookType := tf.Type()
		if hookType.Kind() != reflect.Func {
			continue
		}
		of := ov.Field(i)
		if of.IsNil() {
			continue
		}
		if tf.IsNil() {
			tf.Set(of)
			continue
		}

		// Make a copy of tf for tf to call. (Otherwise it
		// creates a recursive call cycle and stack overflows)
		tfCopy := reflect.ValueOf(tf.Interface())

		// We need to call both tf and of in some order.
		newFunc := reflect.MakeFunc(hookType, func(args []reflect.Value) []reflect.Value {
			tfCopy.Call(args)
			return of.Call(args)
		})
		tv.Field(i).Set(newFunc)
	}
}

// ContextConnTrace returns the ClientTrace associated with the
// provided context. If none, it returns nil.
func ContextHttp3ClientTrace(ctx context.Context) *Http3ClientTrace {
	trace, _ := ctx.Value(http3EventKey{}).(*Http3ClientTrace)
	return trace
}

// WithTrace function binds a specific context.Context to as specific ConnTrace
func WithTrace(ctx context.Context, trace *Http3ClientTrace) context.Context {
	if trace == nil {
		panic("nil trace")
	}
	old := ContextHttp3ClientTrace(ctx)
	trace.compose(old)

	ctx = context.WithValue(ctx, http3EventKey{}, trace)
	return ctx
}

func newHttp3RoundTripper(config *Config, runtimeConfig *RuntimeConfig, w *webClientImpl) (http.RoundTripper, error) {
	if config.Method == http.MethodGet {
		config.Method = http3.MethodGet0RTT
	}
	return &http3.RoundTripper{
		DisableCompression: config.DisableCompression,
		Dial: func(ctx context.Context, addr string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlyConnection, error) {

			trh := httptrace.ContextClientTrace(ctx)

			if trh != nil && trh.GetConn != nil {
				trh.GetConn(addr)
			}

			if trh != nil && trh.DNSStart != nil {
				trh.DNSStart(httptrace.DNSStartInfo{
					Host: addr,
				})
			}

			connAddr, e := w.resolver.resolveConn(addr)

			if e != nil {
				return nil, e
			}
			runtimeConfig.ResolvedConnAddress = connAddr

			if trh != nil && trh.DNSDone != nil {

				trh.DNSDone(httptrace.DNSDoneInfo{
					Addrs: []net.IPAddr{},
				})
			}

			if trh != nil && trh.TLSHandshakeStart != nil {
				trh.TLSHandshakeStart()
			}

			dae, err := quic.DialAddrEarly(ctx, connAddr, tlsCfg, cfg)
			if err != nil {
				return nil, err
			}

			if trh != nil && trh.TLSHandshakeDone != nil {
				trh.TLSHandshakeDone(tls.ConnectionState{}, nil)
			}

			if trh != nil && trh.GotConn != nil {
				trh.GotConn(httptrace.GotConnInfo{Conn: connAdapter{remoteAddr: dae.RemoteAddr()}})
			}

			return wrapEarlyConnection(dae, w), err
		},

		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.NoCheckCertificate,
		},
		QuicConfig: &(quic.Config{}),
	}, nil
}

type connAdapter struct {
	*net.UDPConn
	remoteAddr net.Addr
}

func (c connAdapter) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func CheckAltSvcH3Header(h http.Header) *string {
	for k, entries := range h {
		if strings.ToUpper(k) == "ALT-SVC" {
			for _, entry := range entries {
				if value := CheckAltSvcH3(entry); value != nil {
					return value
				}
			}
		}
	}
	return nil
}

var fieldRx = regexp.MustCompile(`^\s*([a-zA-Z0-9-]+)=(.*)$`)

func CheckAltSvcH3(s string) *string {
	for _, prop := range strings.Split(s, ";") {
		kv := fieldRx.FindStringSubmatch(prop)

		if kv[1] == "h3" {
			vl := kv[2]
			if len(vl) >= 2 && vl[0] == '"' && vl[len(vl)-1] == '"' {
				unquoted := vl[1 : len(vl)-1]
				return &unquoted
			}
			return &vl
		}
	}
	return nil
}
