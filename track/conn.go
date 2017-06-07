// Copyright 2017 Eric Daniels

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//    http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package track

import (
	"net"
	"sync/atomic"
)

// Conn wraps a net.Conn and tracks reads and writes
type Conn interface {
	net.Conn
	ByteTracker
}

// NewConn returns a new Conn based off of a net.Conn
func NewConn(conn net.Conn) Conn {
	return &basicConn{Conn: conn}
}

type basicConn struct {
	net.Conn
	OnClose      func()
	bytesRead    uint64
	bytesWritten uint64
}

func (conn *basicConn) Read(b []byte) (n int, err error) {
	n, err = conn.Conn.Read(b)
	if n > 0 {
		atomic.AddUint64(&conn.bytesRead, uint64(n))
	}
	return n, err
}

func (conn *basicConn) Write(b []byte) (n int, err error) {
	n, err = conn.Conn.Write(b)
	if n > 0 {
		atomic.AddUint64(&conn.bytesWritten, uint64(n))
	}
	return n, err
}

func (conn *basicConn) Close() error {
	if conn.OnClose != nil {
		conn.OnClose()
	}
	return conn.Conn.Close()
}

func (conn *basicConn) BytesRead() uint64 {
	return atomic.LoadUint64(&conn.bytesRead)
}

func (conn *basicConn) BytesWritten() uint64 {
	return atomic.LoadUint64(&conn.bytesWritten)
}

func (conn *basicConn) BytesReadWritten() BytesSummary {
	return BytesSummary{
		Read:    conn.BytesRead(),
		Written: conn.BytesWritten(),
	}
}
