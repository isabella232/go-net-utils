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
	"testing"
	"time"

	gc "github.com/smartystreets/goconvey/convey"
)

func TestDialerReadWriteTracking(t *testing.T) {

	listener := newTestHalfListener(t)

	gc.Convey("Dialer should properly track all bytes transferred at the application layer", t, func() {

		dialer := NewDefaultDialer()
		knownWritten := 0
		knownRead := 0

		for _, payload := range []string{
			"hey look I've got some bytes!",
			"get your bytes here",
			"are you byting to me?",
		} {
			conn, err := dialer.Dial("tcp", listener.Addr().String())
			gc.So(err, gc.ShouldBeNil)
			gc.So(conn, gc.ShouldNotBeNil)

			payloadBytes := []byte(payload)

			for i := 0; i < len(payloadBytes); i += 2 {
				toWrite := 2
				if i+toWrite >= len(payloadBytes) {
					toWrite = 1
				}

				written, err := conn.Write(payloadBytes[i : i+toWrite])
				gc.So(err, gc.ShouldBeNil)
				gc.So(written, gc.ShouldEqual, toWrite)
				knownWritten += written

				bytesRead, bytesWritten := dialer.BytesReadWritten()
				gc.So(bytesRead, gc.ShouldEqual, knownRead)
				gc.So(bytesWritten, gc.ShouldEqual, knownWritten)

				if toWrite/2 == 0 {
					continue
				}
				buff := make([]byte, 64)
				read, err := conn.Read(buff)
				gc.So(err, gc.ShouldBeNil)
				gc.So(read, gc.ShouldEqual, toWrite/2)
				knownRead += read
			}

			conn.Close()

			read, written := dialer.BytesReadWritten()
			gc.So(read, gc.ShouldEqual, knownRead)
			gc.So(written, gc.ShouldEqual, knownWritten)
		}
	})
}

func TestDialerContextReadWriteTracking(t *testing.T) {

	listener := newTestHalfListener(t)

	gc.Convey("Dialer should properly track all bytes transferred at the application layer", t, func() {

		dialer := NewDefaultDialer()
		knownWritten := 0
		knownRead := 0

		for _, payload := range []string{
			"hey look I've got some bytes!",
			"get your bytes here",
			"are you byting to me?",
		} {
			conn, err := dialer.DialContext(context.Background(), "tcp", listener.Addr().String())
			gc.So(err, gc.ShouldBeNil)
			gc.So(conn, gc.ShouldNotBeNil)

			payloadBytes := []byte(payload)

			for i := 0; i < len(payloadBytes); i += 2 {
				toWrite := 2
				if i+toWrite >= len(payloadBytes) {
					toWrite = 1
				}

				written, err := conn.Write(payloadBytes[i : i+toWrite])
				gc.So(err, gc.ShouldBeNil)
				gc.So(written, gc.ShouldEqual, toWrite)
				knownWritten += written

				bytesRead, bytesWritten := dialer.BytesReadWritten()
				gc.So(bytesRead, gc.ShouldEqual, knownRead)
				gc.So(bytesWritten, gc.ShouldEqual, knownWritten)

				if toWrite/2 == 0 {
					continue
				}
				buff := make([]byte, 64)
				read, err := conn.Read(buff)
				gc.So(err, gc.ShouldBeNil)
				gc.So(read, gc.ShouldEqual, toWrite/2)
				knownRead += read
			}

			conn.Close()

			read, written := dialer.BytesReadWritten()
			gc.So(read, gc.ShouldEqual, knownRead)
			gc.So(written, gc.ShouldEqual, knownWritten)
		}

		dialer.ResetBytes()

		bytesRead, bytesWritten := dialer.BytesReadWritten()
		gc.So(bytesRead, gc.ShouldEqual, 0)
		gc.So(bytesWritten, gc.ShouldEqual, 0)
	})
}

func TestDialerGetReadWriteAndClose(t *testing.T) {
	gc.Convey("Dialer should allow getting reads and writes while a connection is closing", t, func() {

		listener, err := net.Listen("tcp", "localhost:0")
		gc.So(err, gc.ShouldBeNil)
		defer listener.Close()

		// 1. Accept two connections.
		go func() {
			_, err := listener.Accept()
			if err != nil {
				panic(err)
			}

			_, err = listener.Accept()
			if err != nil {
				panic(err)
			}
		}()

		connReadStartCh := make(chan struct{}, 2)

		dialer := &basicDialer{
			netDialer: testNetDialer{
				d: &net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 5 * time.Second,
					DualStack: true,
				},
				ConnReadStart: func() {
					connReadStartCh <- struct{}{}
				},
			},
		}

		bytesReadWrittenStart := make(chan int, 2)

		// 2. Get two connections.
		conn1, err := dialer.Dial("tcp", listener.Addr().String())
		gc.So(err, gc.ShouldBeNil)

		var bytesReadWrittenStartOnce1 sync.Once
		conn1.(*basicConn).onBytesReadWrittenStart = func() {
			bytesReadWrittenStartOnce1.Do(func() {
				bytesReadWrittenStart <- 1
			})
		}

		conn2, err := dialer.Dial("tcp", listener.Addr().String())
		gc.So(err, gc.ShouldBeNil)

		var bytesReadWrittenStartOnce2 sync.Once
		conn2.(*basicConn).onBytesReadWrittenStart = func() {
			bytesReadWrittenStartOnce2.Do(func() {
				bytesReadWrittenStart <- 2
			})
		}

		readErrCh := make(chan error, 2)

		// 3. Start reads.
		go func() {
			buff := make([]byte, 64)
			_, err := conn1.Read(buff)
			readErrCh <- err
		}()

		go func() {
			buff := make([]byte, 64)
			_, err := conn2.Read(buff)
			readErrCh <- err
		}()

		// 4. Wait for read starting on both connections.
		<-connReadStartCh
		<-connReadStartCh

		// 5. Get bytes read and written from the dialer.
		resultCh := make(chan struct{})
		go func() {
			dialer.BytesReadWritten()
			resultCh <- struct{}{}
		}()

		first := <-bytesReadWrittenStart

		// 6. Close the opposite connection.
		closeDoneCh := make(chan struct{})
		go func(firstConn int) {
			if firstConn == 1 {
				conn2.Close()
			} else {
				conn1.Close()
			}
			closeDoneCh <- struct{}{}
		}(first)

		// A deadlock would happen here after 6 if anything.
		// Wait a bit before failing test. This would be if
		// closing one connection that is not the one being read
		// causes a cycle between the closing goroutine and the
		// bytes tracking reader goroutine while a lock for both
		// is being held.
		timer := time.NewTimer(time.Second * 5)
		select {
		case <-timer.C:
			t.Error("Deadlock")
			t.FailNow()
		case <-closeDoneCh:
			timer.Stop()
		}

		<-bytesReadWrittenStart

		// 7. Close the connection opposite from the first close.
		if first == 1 {
			conn1.Close()
		} else {
			conn2.Close()
		}

		// 8. Wait for BytesReadWritten done.
		<-resultCh

		// 9. readErr should indicate read closed.
		readErr := <-readErrCh
		gc.So(readErr, gc.ShouldNotBeNil)
		readErr = <-readErrCh
		gc.So(readErr, gc.ShouldNotBeNil)
	})
}
