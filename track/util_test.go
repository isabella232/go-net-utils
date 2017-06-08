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
	"testing"
)

//	newTestHalfListener will write back half of the bytes it reads
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
