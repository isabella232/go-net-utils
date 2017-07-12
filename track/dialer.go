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
	bytesRead    uint64
	bytesWritten uint64
	connCounter  uint64

	*net.Dialer
	conns    map[uint64]Conn
	connsMut sync.RWMutex

	// flushMut is for critical sections that can
	// affect totals
	flushMut sync.RWMutex

	dialMut sync.Mutex
}

// NewDefaultDialer returns a Dialer based on
// a default net.Dialer
func NewDefaultDialer() Dialer {
	return &basicDialer{
		Dialer: &net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 5 * time.Second,
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
	dialer.dialMut.Lock()
	conn, err := dialer.Dialer.Dial(network, address)
	if err != nil {
		dialer.dialMut.Unlock()
		return nil, err
	}
	dialer.dialMut.Unlock()

	connNum := dialer.nextConnNum()
	tConn := newConn(conn)
	tConn.OnClose = dialer.makeOnConnClose(connNum)
	dialer.addConn(tConn, connNum)

	return tConn, nil
}

func (dialer *basicDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	dialer.dialMut.Lock()
	conn, err := dialer.Dialer.DialContext(ctx, network, address)
	if err != nil {
		dialer.dialMut.Unlock()
		return nil, err
	}
	dialer.dialMut.Unlock()

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
	defer dialer.flushMut.RUnlock()

	var totalRead, totalWritten uint64

	dialer.connsMut.RLock()
	for _, conn := range dialer.conns {
		read, written := conn.BytesReadWritten()
		totalRead += read
		totalWritten += written
	}
	dialer.connsMut.RUnlock()

	totalRead += atomic.LoadUint64(&dialer.bytesRead)
	totalWritten += atomic.LoadUint64(&dialer.bytesWritten)

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

func (dialer *basicDialer) Timeout() time.Duration {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	return dialer.Dialer.Timeout
}

func (dialer *basicDialer) Deadline() time.Time {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	return dialer.Dialer.Deadline
}

func (dialer *basicDialer) LocalAddr() net.Addr {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	return dialer.Dialer.LocalAddr
}

func (dialer *basicDialer) DualStack() bool {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	return dialer.Dialer.DualStack
}

func (dialer *basicDialer) FallbackDelay() time.Duration {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	return dialer.Dialer.FallbackDelay
}

func (dialer *basicDialer) KeepAlive() time.Duration {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	return dialer.Dialer.KeepAlive
}

func (dialer *basicDialer) Resolver() *net.Resolver {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	return dialer.Dialer.Resolver
}

func (dialer *basicDialer) SetTimeout(newTimeout time.Duration) {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	dialer.Dialer.Timeout = newTimeout
}

func (dialer *basicDialer) SetDeadline(newDeadline time.Time) {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	dialer.Dialer.Deadline = newDeadline
}

func (dialer *basicDialer) SetLocalAddr(newLocalAddr net.Addr) {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	dialer.Dialer.LocalAddr = newLocalAddr
}

func (dialer *basicDialer) SetDualStack(newDualStack bool) {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	dialer.Dialer.DualStack = newDualStack
}

func (dialer *basicDialer) SetFallbackDelay(newFallbackDelay time.Duration) {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	dialer.Dialer.FallbackDelay = newFallbackDelay
}

func (dialer *basicDialer) SetKeepAlive(newKeepAlive time.Duration) {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	dialer.Dialer.KeepAlive = newKeepAlive
}

func (dialer *basicDialer) SetResolver(newResolver *net.Resolver) {
	dialer.dialMut.Lock()
	defer dialer.dialMut.Unlock()
	dialer.Dialer.Resolver = newResolver
}
