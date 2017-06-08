// Copyright 2017 Eric Daniels
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package track

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Dialer represents a net.Dialer that can track bytes read/written
type Dialer interface {
	ByteTracker

	Dial(network, address string) (net.Conn, error)
	DialContext(ctx context.Context, network, address string) (net.Conn, error)

	// Accessors
	Timeout() time.Duration
	Deadline() time.Time
	LocalAddr() net.Addr
	DualStack() bool
	FallbackDelay() time.Duration
	KeepAlive() time.Duration
	Resolver() *net.Resolver

	// Mutators
	SetTimeout(newTimeout time.Duration)
	SetDeadline(newDeadline time.Time)
	SetLocalAddr(newLocalAddr net.Addr)
	SetDualStack(newDualStack bool)
	SetFallbackDelay(newFallbackDelay time.Duration)
	SetKeepAlive(newKeepAlive time.Duration)
	SetResolver(newResolver *net.Resolver)
}

type basicDialer struct {
	*net.Dialer
	connCounter  uint64
	conns        map[uint64]Conn
	connsMut     sync.RWMutex
	bytesRead    uint64
	bytesWritten uint64
}

// NewDefaultDialer returns a Dialer based on
// a default net.Dialer
func NewDefaultDialer() Dialer {
	return &basicDialer{
		Dialer: &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		},
	}
}

func (dialer *basicDialer) addConn(conn Conn, num uint64) {
	dialer.connsMut.Lock()
	if dialer.conns == nil {
		dialer.conns = map[uint64]Conn{}
	}
	dialer.conns[num] = conn
	dialer.connsMut.Unlock()
}

func (dialer *basicDialer) Dial(network, address string) (net.Conn, error) {
	conn, err := dialer.Dialer.Dial(network, address)
	if err != nil {
		return nil, err
	}
	connNum := dialer.nextConnNum()
	tConn := &basicConn{Conn: conn, OnClose: dialer.makeOnConnClose(connNum)}
	dialer.addConn(tConn, connNum)

	return tConn, nil
}

func (dialer *basicDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := dialer.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	connNum := dialer.nextConnNum()
	tConn := &basicConn{Conn: conn, OnClose: dialer.makeOnConnClose(connNum)}
	dialer.addConn(tConn, connNum)

	return tConn, nil
}

func (dialer *basicDialer) makeOnConnClose(connNum uint64) func() {
	return func() {
		dialer.connsMut.Lock()

		// Ensure connection exists for duration of this
		// call until we delete the connection
		conn, ok := dialer.conns[connNum]

		if !ok {
			dialer.connsMut.Unlock()
			return
		}
		delete(dialer.conns, connNum)
		dialer.connsMut.Unlock()

		summary := conn.BytesReadWritten()
		atomic.AddUint64(&dialer.bytesRead, summary.Read)
		atomic.AddUint64(&dialer.bytesWritten, summary.Written)
	}
}

func (dialer *basicDialer) nextConnNum() uint64 {
	return atomic.AddUint64(&dialer.connCounter, 1)
}

func (dialer *basicDialer) BytesRead() uint64 {
	return dialer.BytesReadWritten().Read
}

func (dialer *basicDialer) BytesWritten() uint64 {
	return dialer.BytesReadWritten().Written
}

func (dialer *basicDialer) BytesReadWritten() BytesSummary {
	var totalRead, totalWritten uint64

	dialer.connsMut.RLock()
	for _, conn := range dialer.conns {
		totalRead += conn.BytesRead()
		totalWritten += conn.BytesWritten()
	}
	dialer.connsMut.RUnlock()

	totalRead += atomic.LoadUint64(&dialer.bytesRead)
	totalWritten += atomic.LoadUint64(&dialer.bytesWritten)

	return BytesSummary{Read: totalRead, Written: totalWritten}
}

func (dialer *basicDialer) ResetBytes() {
	dialer.connsMut.Lock()
	for _, conn := range dialer.conns {
		conn.ResetBytes()
	}
	dialer.connsMut.Unlock()

	atomic.StoreUint64(&dialer.bytesRead, 0)
	atomic.StoreUint64(&dialer.bytesWritten, 0)
}

func (dialer *basicDialer) Timeout() time.Duration {
	return dialer.Dialer.Timeout
}

func (dialer *basicDialer) Deadline() time.Time {
	return dialer.Dialer.Deadline
}

func (dialer *basicDialer) LocalAddr() net.Addr {
	return dialer.Dialer.LocalAddr
}

func (dialer *basicDialer) DualStack() bool {
	return dialer.Dialer.DualStack
}

func (dialer *basicDialer) FallbackDelay() time.Duration {
	return dialer.Dialer.FallbackDelay
}

func (dialer *basicDialer) KeepAlive() time.Duration {
	return dialer.Dialer.KeepAlive
}

func (dialer *basicDialer) Resolver() *net.Resolver {
	return dialer.Dialer.Resolver
}

func (dialer *basicDialer) SetTimeout(newTimeout time.Duration) {
	dialer.Dialer.Timeout = newTimeout
}

func (dialer *basicDialer) SetDeadline(newDeadline time.Time) {
	dialer.Dialer.Deadline = newDeadline
}

func (dialer *basicDialer) SetLocalAddr(newLocalAddr net.Addr) {
	dialer.Dialer.LocalAddr = newLocalAddr
}

func (dialer *basicDialer) SetDualStack(newDualStack bool) {
	dialer.Dialer.DualStack = newDualStack
}

func (dialer *basicDialer) SetFallbackDelay(newFallbackDelay time.Duration) {
	dialer.Dialer.FallbackDelay = newFallbackDelay
}

func (dialer *basicDialer) SetKeepAlive(newKeepAlive time.Duration) {
	dialer.Dialer.KeepAlive = newKeepAlive
}

func (dialer *basicDialer) SetResolver(newResolver *net.Resolver) {
	dialer.Dialer.Resolver = newResolver
}
