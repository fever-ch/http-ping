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
	"github.com/quic-go/quic-go"
	"net"
	"sync/atomic"
	"time"
)

func wrapEarlyConnection(ec quic.EarlyConnection, w *webClientImpl) quic.EarlyConnection {
	return &earlyConnectionWrapper{ec: ec, webClientImpl: w}
}

func wrapStream(st quic.Stream, w *webClientImpl) quic.Stream {
	return &streamWrapper{stream: st, webClientImpl: w}
}

type earlyConnectionWrapper struct {
	ec            quic.EarlyConnection
	webClientImpl *webClientImpl
}

func (earlyConnectionWrapper *earlyConnectionWrapper) AcceptStream(ctx context.Context) (quic.Stream, error) {
	s, er := earlyConnectionWrapper.ec.AcceptStream(ctx)
	return wrapStream(s, earlyConnectionWrapper.webClientImpl), er
}

func (earlyConnectionWrapper *earlyConnectionWrapper) AcceptUniStream(ctx context.Context) (quic.ReceiveStream, error) {
	return earlyConnectionWrapper.ec.AcceptUniStream(ctx)
}

func (earlyConnectionWrapper *earlyConnectionWrapper) OpenStream() (quic.Stream, error) {
	return earlyConnectionWrapper.ec.OpenStream()
}

func (earlyConnectionWrapper *earlyConnectionWrapper) OpenStreamSync(ctx context.Context) (quic.Stream, error) {
	s, er := earlyConnectionWrapper.ec.OpenStreamSync(ctx)
	return wrapStream(s, earlyConnectionWrapper.webClientImpl), er
}

func (earlyConnectionWrapper *earlyConnectionWrapper) OpenUniStream() (quic.SendStream, error) {
	return earlyConnectionWrapper.ec.OpenUniStream()
}

func (earlyConnectionWrapper *earlyConnectionWrapper) OpenUniStreamSync(ctx context.Context) (quic.SendStream, error) {
	return earlyConnectionWrapper.ec.OpenUniStreamSync(ctx)
}

func (earlyConnectionWrapper *earlyConnectionWrapper) LocalAddr() net.Addr {
	return earlyConnectionWrapper.ec.LocalAddr()
}

func (earlyConnectionWrapper *earlyConnectionWrapper) RemoteAddr() net.Addr {
	return earlyConnectionWrapper.ec.RemoteAddr()
}

func (earlyConnectionWrapper *earlyConnectionWrapper) CloseWithError(code quic.ApplicationErrorCode, s string) error {
	return earlyConnectionWrapper.ec.CloseWithError(code, s)
}

func (earlyConnectionWrapper *earlyConnectionWrapper) Context() context.Context {
	return earlyConnectionWrapper.ec.Context()
}

func (earlyConnectionWrapper *earlyConnectionWrapper) ConnectionState() quic.ConnectionState {
	return earlyConnectionWrapper.ec.ConnectionState()
}

func (earlyConnectionWrapper *earlyConnectionWrapper) SendMessage(bytes []byte) error {
	return earlyConnectionWrapper.ec.SendMessage(bytes)
}

func (earlyConnectionWrapper *earlyConnectionWrapper) ReceiveMessage(context context.Context) ([]byte, error) {
	return earlyConnectionWrapper.ec.ReceiveMessage(context)
}

func (earlyConnectionWrapper *earlyConnectionWrapper) HandshakeComplete() <-chan struct{} {
	return earlyConnectionWrapper.ec.HandshakeComplete()
}

func (earlyConnectionWrapper *earlyConnectionWrapper) NextConnection() quic.Connection {
	return earlyConnectionWrapper.ec.NextConnection()
}

type streamWrapper struct {
	stream        quic.Stream
	webClientImpl *webClientImpl
}

func (streamWrapper *streamWrapper) StreamID() quic.StreamID {
	return streamWrapper.stream.StreamID()
}

func (streamWrapper *streamWrapper) Read(p []byte) (int, error) {
	n, err := streamWrapper.stream.Read(p)
	atomic.AddInt64(&streamWrapper.webClientImpl.reads, int64(n))
	return n, err
}

func (streamWrapper *streamWrapper) CancelRead(code quic.StreamErrorCode) {
	streamWrapper.stream.CancelRead(code)
}

func (streamWrapper *streamWrapper) SetReadDeadline(t time.Time) error {
	return streamWrapper.stream.SetReadDeadline(t)
}

func (streamWrapper *streamWrapper) Write(p []byte) (int, error) {
	atomic.AddInt64(&streamWrapper.webClientImpl.writes, int64(len(p)))

	return streamWrapper.stream.Write(p)
}

func (streamWrapper *streamWrapper) Close() error {
	return streamWrapper.stream.Close()
}

func (streamWrapper *streamWrapper) CancelWrite(code quic.StreamErrorCode) {
	streamWrapper.stream.CancelWrite(code)
}

func (streamWrapper *streamWrapper) Context() context.Context {
	return streamWrapper.stream.Context()
}

func (streamWrapper *streamWrapper) SetWriteDeadline(t time.Time) error {
	return streamWrapper.stream.SetWriteDeadline(t)
}

func (streamWrapper *streamWrapper) SetDeadline(t time.Time) error {
	return streamWrapper.stream.SetDeadline(t)
}
