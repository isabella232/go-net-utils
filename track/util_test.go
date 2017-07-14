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
	"testing"
	"time"
)

// newTestHalfListener will write back half of the bytes it reads
func newTestHalfListener(t *testing.T) net.Listener {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Logf("error listening: %v", err)
		t.FailNow()
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				continue
			}

			go func() {
				buff := make([]byte, 4)
				for {
					rd, err := conn.Read(buff)
					if err != nil {
						return
					}

					_, err = conn.Write(buff[:rd/2])
					if err != nil {
						return
					}
				}
			}()
		}
	}()

	return listener
}

type testNetConn struct {
	net.Conn
	ReadStart  func()
	WriteStart func()
}

func (conn testNetConn) Read(b []byte) (n int, err error) {
	if conn.ReadStart != nil {
		conn.ReadStart()
	}
	return conn.Conn.Read(b)
}

func (conn testNetConn) Write(b []byte) (n int, err error) {
	if conn.WriteStart != nil {
		conn.WriteStart()
	}
	return conn.Conn.Write(b)
}

type testNetDialer struct {
	d              *net.Dialer
	ConnReadStart  func()
	ConnWriteStart func()
}

func (dialer testNetDialer) Dial(network, address string) (net.Conn, error) {
	conn, err := dialer.d.Dial(network, address)
	if err != nil {
		return nil, err
	}

	return testNetConn{
		Conn:       conn,
		ReadStart:  dialer.ConnReadStart,
		WriteStart: dialer.ConnWriteStart,
	}, nil
}

func (dialer testNetDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := dialer.d.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	return testNetConn{
		Conn:       conn,
		ReadStart:  dialer.ConnReadStart,
		WriteStart: dialer.ConnWriteStart,
	}, nil
}

func (dialer testNetDialer) Timeout() time.Duration {
	return dialer.d.Timeout
}

func (dialer testNetDialer) Deadline() time.Time {
	return dialer.d.Deadline
}

func (dialer testNetDialer) LocalAddr() net.Addr {
	return dialer.d.LocalAddr
}

func (dialer testNetDialer) DualStack() bool {
	return dialer.d.DualStack
}

func (dialer testNetDialer) FallbackDelay() time.Duration {
	return dialer.d.FallbackDelay
}

func (dialer testNetDialer) KeepAlive() time.Duration {
	return dialer.d.KeepAlive
}

func (dialer testNetDialer) Resolver() *net.Resolver {
	return dialer.d.Resolver
}

func (dialer testNetDialer) SetTimeout(newTimeout time.Duration) {
	dialer.d.Timeout = newTimeout
}

func (dialer testNetDialer) SetDeadline(newDeadline time.Time) {
	dialer.d.Deadline = newDeadline
}

func (dialer testNetDialer) SetLocalAddr(newLocalAddr net.Addr) {
	dialer.d.LocalAddr = newLocalAddr
}

func (dialer testNetDialer) SetDualStack(newDualStack bool) {
	dialer.d.DualStack = newDualStack
}

func (dialer testNetDialer) SetFallbackDelay(newFallbackDelay time.Duration) {
	dialer.d.FallbackDelay = newFallbackDelay
}

func (dialer testNetDialer) SetKeepAlive(newKeepAlive time.Duration) {
	dialer.d.KeepAlive = newKeepAlive
}

func (dialer testNetDialer) SetResolver(newResolver *net.Resolver) {
	dialer.d.Resolver = newResolver
}
