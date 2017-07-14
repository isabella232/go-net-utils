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
	netDialer
}

type netDialer interface {
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
	bytesRead    uint64
	bytesWritten uint64
	connCounter  uint64

	netDialer
	conns    map[uint64]Conn
	connsMut sync.RWMutex

	// flushMut is for critical sections that can
	// affect totals
	flushMut sync.RWMutex
}

// NewDefaultDialer returns a Dialer based on
// a default net.Dialer
func NewDefaultDialer() Dialer {
	return &basicDialer{
		netDialer: netDialerWrapper{
			d: &net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 5 * time.Second,
				DualStack: true,
			},
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
	conn, err := dialer.netDialer.Dial(network, address)
	if err != nil {
		return nil, err
	}

	connNum := dialer.nextConnNum()
	tConn := newConn(conn)
	tConn.OnClose = dialer.makeOnConnClose(connNum)
	dialer.addConn(tConn, connNum)

	return tConn, nil
}

func (dialer *basicDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := dialer.netDialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	connNum := dialer.nextConnNum()
	tConn := newConn(conn)
	tConn.OnClose = dialer.makeOnConnClose(connNum)
	dialer.addConn(tConn, connNum)

	return tConn, nil
}

func (dialer *basicDialer) makeOnConnClose(connNum uint64) func() {
	return func() {
		dialer.flushMut.Lock()
		defer dialer.flushMut.Unlock()

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

		read, written := conn.BytesReadWritten()
		atomic.AddUint64(&dialer.bytesRead, read)
		atomic.AddUint64(&dialer.bytesWritten, written)
	}
}

func (dialer *basicDialer) nextConnNum() uint64 {
	return atomic.AddUint64(&dialer.connCounter, 1)
}

func (dialer *basicDialer) BytesRead() uint64 {
	read, _ := dialer.BytesReadWritten()
	return read
}

func (dialer *basicDialer) BytesWritten() uint64 {
	_, written := dialer.BytesReadWritten()
	return written
}

func (dialer *basicDialer) BytesReadWritten() (uint64, uint64) {

	dialer.flushMut.RLock()

	// Capture bytesRead and bytesWritten before a Conn on close
	// handler gets called
	totalRead := atomic.LoadUint64(&dialer.bytesRead)
	totalWritten := atomic.LoadUint64(&dialer.bytesWritten)

	// Shadow copy of conns such that conns is the state
	// before any connection closes and is deleted from
	// conns
	dialer.connsMut.RLock()
	shadowConns := make([]Conn, 0, len(dialer.conns))
	for _, conn := range dialer.conns {
		shadowConns = append(shadowConns, conn)
	}
	dialer.connsMut.RUnlock()

	dialer.flushMut.RUnlock()

	for _, conn := range shadowConns {
		read, written := conn.BytesReadWritten()
		totalRead += read
		totalWritten += written
	}

	return totalRead, totalWritten
}

func (dialer *basicDialer) ResetBytes() {
	dialer.flushMut.Lock()
	defer dialer.flushMut.Unlock()

	dialer.connsMut.Lock()
	for _, conn := range dialer.conns {
		conn.ResetBytes()
	}
	dialer.connsMut.Unlock()

	atomic.StoreUint64(&dialer.bytesRead, 0)
	atomic.StoreUint64(&dialer.bytesWritten, 0)
}

type netDialerWrapper struct {
	d       *net.Dialer
	dialMut sync.Mutex
}

func (dialer netDialerWrapper) Dial(network, address string) (net.Conn, error) {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	return dialer.d.Dial(network, address)
}

func (dialer netDialerWrapper) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	return dialer.d.DialContext(ctx, network, address)
}

func (dialer netDialerWrapper) Timeout() time.Duration {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	return dialer.d.Timeout
}

func (dialer netDialerWrapper) Deadline() time.Time {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	return dialer.d.Deadline
}

func (dialer netDialerWrapper) LocalAddr() net.Addr {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	return dialer.d.LocalAddr
}

func (dialer netDialerWrapper) DualStack() bool {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	return dialer.d.DualStack
}

func (dialer netDialerWrapper) FallbackDelay() time.Duration {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	return dialer.d.FallbackDelay
}

func (dialer netDialerWrapper) KeepAlive() time.Duration {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	return dialer.d.KeepAlive
}

func (dialer netDialerWrapper) Resolver() *net.Resolver {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	return dialer.d.Resolver
}

func (dialer netDialerWrapper) SetTimeout(newTimeout time.Duration) {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	dialer.d.Timeout = newTimeout
}

func (dialer netDialerWrapper) SetDeadline(newDeadline time.Time) {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	dialer.d.Deadline = newDeadline
}

func (dialer netDialerWrapper) SetLocalAddr(newLocalAddr net.Addr) {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	dialer.d.LocalAddr = newLocalAddr
}

func (dialer netDialerWrapper) SetDualStack(newDualStack bool) {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	dialer.d.DualStack = newDualStack
}

func (dialer netDialerWrapper) SetFallbackDelay(newFallbackDelay time.Duration) {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	dialer.d.FallbackDelay = newFallbackDelay
}

func (dialer netDialerWrapper) SetKeepAlive(newKeepAlive time.Duration) {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	dialer.d.KeepAlive = newKeepAlive
}

func (dialer netDialerWrapper) SetResolver(newResolver *net.Resolver) {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	dialer.d.Resolver = newResolver
}
