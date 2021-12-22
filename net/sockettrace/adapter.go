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

package sockettrace

import (
	"golang.org/x/net/context"
	"net"
	"reflect"
	"time"
)

type socketTraceEventContextKey struct{}

// ConnTrace is a set of hooks to run at various events of a TCP connection
type ConnTrace struct {
	Write          func(size int)
	Read           func(size int)
	TCPStart       func()
	TCPEstablished func()
}

type connAdapter struct {
	innerConn net.Conn
	connTrace *ConnTrace
}

// ContextConnTrace returns the ClientTrace associated with the
// provided context. If none, it returns nil.
func ContextConnTrace(ctx context.Context) *ConnTrace {
	trace, _ := ctx.Value(socketTraceEventContextKey{}).(*ConnTrace)
	return trace
}

// WithTrace function binds a specific context.Context to as specific ConnTrace
func WithTrace(ctx context.Context, trace *ConnTrace) context.Context {
	if trace == nil {
		panic("nil trace")
	}
	old := ContextConnTrace(ctx)
	trace.compose(old)

	ctx = context.WithValue(ctx, socketTraceEventContextKey{}, trace)
	return ctx
}

// NewSocketTrace generates a new net.Conn which generates statistics for the current context
func NewSocketTrace(context context.Context, dialer *net.Dialer, network, ipaddr string) (net.Conn, error) {
	socketTraceSocketEventContext, _ := context.Value(socketTraceEventContextKey{}).(*ConnTrace)

	if socketTraceSocketEventContext.TCPStart != nil {
		socketTraceSocketEventContext.TCPStart()
	}

	conn, err := dialer.DialContext(context, network, ipaddr)

	if socketTraceSocketEventContext.TCPEstablished != nil {
		socketTraceSocketEventContext.TCPEstablished()
	}

	if err != nil {
		return nil, err
	}

	return &connAdapter{
		innerConn: conn,
		connTrace: socketTraceSocketEventContext,
	}, nil
}

// compose modifies t such that it respects the previously-registered hooks in old,
// subject to the composition policy requested in t.Compose.
func (t *ConnTrace) compose(old *ConnTrace) {
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

// Read behaves is a proxy to the actual conn.Read (counts reads)
func (sta *connAdapter) Read(b []byte) (int, error) {
	n, err := sta.innerConn.Read(b)
	if sta.connTrace != nil && sta.connTrace.Read != nil {
		sta.connTrace.Read(n)
	}
	return n, err
}

// Write behaves is a proxy to the actual conn.Write (counts writes)
func (sta *connAdapter) Write(b []byte) (int, error) {
	n, err := sta.innerConn.Write(b)
	if sta.connTrace != nil && sta.connTrace.Write != nil {
		sta.connTrace.Write(n)
	}
	return n, err
}

// Close behaves is a proxy to the actual conn.Close
func (sta *connAdapter) Close() error {
	return sta.innerConn.Close()
}

// LocalAddr behaves is a proxy to the actual conn.LocalAddr
func (sta *connAdapter) LocalAddr() net.Addr {
	return sta.innerConn.LocalAddr()
}

// RemoteAddr behaves is a proxy to the actual conn.RemoteAddr
func (sta *connAdapter) RemoteAddr() net.Addr {
	return sta.innerConn.RemoteAddr()
}

// SetDeadline behaves is a proxy to the actual conn.SetDeadline
func (sta *connAdapter) SetDeadline(t time.Time) error {
	return sta.innerConn.SetDeadline(t)
}

// SetReadDeadline behaves is a proxy to the actual conn.SetReadDeadline
func (sta *connAdapter) SetReadDeadline(t time.Time) error {
	return sta.innerConn.SetReadDeadline(t)
}

// SetWriteDeadline behaves is a proxy to the actual conn.SetWriteDeadline
func (sta *connAdapter) SetWriteDeadline(t time.Time) error {
	return sta.innerConn.SetWriteDeadline(t)
}
