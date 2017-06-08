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
	"testing"

	gc "github.com/smartystreets/goconvey/convey"
)

func TestConnReadWriteTracking(t *testing.T) {

	listener := newTestHalfListener(t)

	gc.Convey("Conn should properly track all bytes transferred at the application layer", t, func() {
		conn, err := net.Dial("tcp", listener.Addr().String())
		gc.So(err, gc.ShouldBeNil)
		gc.So(conn, gc.ShouldNotBeNil)

		tConn := NewConn(conn)
		gc.So(tConn.BytesRead(), gc.ShouldEqual, 0)
		gc.So(tConn.BytesWritten(), gc.ShouldEqual, 0)

		payload := []byte("hey look I've got some bytes!")

		for i := 0; i < len(payload); i += 2 {
			toWrite := 2
			if i+toWrite >= len(payload) {
				toWrite = 1
			}

			written, err := tConn.Write(payload[i : i+toWrite])
			gc.So(err, gc.ShouldBeNil)
			gc.So(written, gc.ShouldEqual, toWrite)
			gc.So(tConn.BytesRead(), gc.ShouldEqual, i/2)
			gc.So(tConn.BytesWritten(), gc.ShouldEqual, i+toWrite)

			if toWrite/2 == 0 {
				continue
			}
			buff := make([]byte, 64)
			read, err := tConn.Read(buff)
			gc.So(err, gc.ShouldBeNil)
			gc.So(read, gc.ShouldEqual, toWrite/2)
		}

		tConn.Close()

		summary := tConn.BytesReadWritten()
		gc.So(summary.Read, gc.ShouldEqual, 14)
		gc.So(summary.Written, gc.ShouldEqual, 29)

		tConn.BytesReset()
		gc.So(tConn.BytesRead(), gc.ShouldEqual, 0)
		gc.So(tConn.BytesWritten(), gc.ShouldEqual, 0)
	})
}
