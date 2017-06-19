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
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Conn wraps a net.Conn and tracks reads and writes
type Conn interface {
	net.Conn
	ByteTracker
}

// newConn returns a new Conn based off of a net.Conn
func newConn(conn net.Conn) *basicConn {
	// Must set a deadline otherwise we risk
	// waiting forever on observation
	conn.SetReadDeadline(time.Now().Add(time.Second * 60))
	conn.SetWriteDeadline(time.Now().Add(time.Second * 60))
	return &basicConn{Conn: conn}
}

// NewConn returns a new Conn based off of a net.Conn
func NewConn(conn net.Conn) Conn {
	return newConn(conn)
}

type basicConn struct {
	bytesRead    uint64
	bytesWritten uint64
	net.Conn
	OnClose   func()
	activeOps sync.WaitGroup
}

func (conn *basicConn) Read(b []byte) (n int, err error) {
	conn.activeOps.Add(1)
	n, err = conn.Conn.Read(b)
	if n > 0 {
		atomic.AddUint64(&conn.bytesRead, uint64(n))
	}
	conn.activeOps.Done()
	conn.SetReadDeadline(time.Now().Add(time.Second * 60))
	return n, err
}

func (conn *basicConn) Write(b []byte) (n int, err error) {
	conn.activeOps.Add(1)
	n, err = conn.Conn.Write(b)
	if n > 0 {
		atomic.AddUint64(&conn.bytesWritten, uint64(n))
	}
	conn.activeOps.Done()
	conn.SetWriteDeadline(time.Now().Add(time.Second * 60))
	return n, err
}

func (conn *basicConn) Close() error {
	err := conn.Conn.Close()
	if conn.OnClose != nil {
		conn.OnClose()
	}
	return err
}

func (conn *basicConn) BytesRead() uint64 {
	return atomic.LoadUint64(&conn.bytesRead)
}

func (conn *basicConn) BytesWritten() uint64 {
	return atomic.LoadUint64(&conn.bytesWritten)
}

func (conn *basicConn) BytesReadWritten() (uint64, uint64) {
	conn.activeOps.Wait()
	return conn.BytesRead(), conn.BytesWritten()
}

func (conn *basicConn) ResetBytes() {
	atomic.StoreUint64(&conn.bytesRead, 0)
	atomic.StoreUint64(&conn.bytesWritten, 0)
}
