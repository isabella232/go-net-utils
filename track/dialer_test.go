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
	"context"
	"testing"

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
				gc.So(dialer.BytesRead(), gc.ShouldEqual, knownRead)
				gc.So(dialer.BytesWritten(), gc.ShouldEqual, knownWritten)

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

			summary := dialer.BytesReadWritten()
			gc.So(summary.Read, gc.ShouldEqual, knownRead)
			gc.So(summary.Written, gc.ShouldEqual, knownWritten)
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
				gc.So(dialer.BytesRead(), gc.ShouldEqual, knownRead)
				gc.So(dialer.BytesWritten(), gc.ShouldEqual, knownWritten)

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

			summary := dialer.BytesReadWritten()
			gc.So(summary.Read, gc.ShouldEqual, knownRead)
			gc.So(summary.Written, gc.ShouldEqual, knownWritten)
		}

		dialer.ResetBytes()
		gc.So(dialer.BytesRead(), gc.ShouldEqual, 0)
		gc.So(dialer.BytesWritten(), gc.ShouldEqual, 0)
	})
}
