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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	gc "github.com/smartystreets/goconvey/convey"
)

func TestHTTPClientRequestTracking(t *testing.T) {

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"some": "otherCoolStuff"}`))
	}))
	defer testServer.Close()

	gc.Convey("Client should properly track all bytes transferred for an HTTP request", t, func() {
		client := NewDefaultHTTPClient()
		defer client.CloseIdleConnections()

		resp, err := client.Get(testServer.URL)
		gc.So(err, gc.ShouldBeNil)
		ioutil.ReadAll(resp.Body)
		gc.So(resp.Body.Close(), gc.ShouldBeNil)
		client.CloseIdleConnections()

		bytesRead, bytesWritten := client.BytesReadWritten()
		gc.So(bytesRead, gc.ShouldEqual, 134)
		gc.So(bytesWritten, gc.ShouldEqual, 96)
	})

	gc.Convey("Client should properly track all bytes transferred for an HTTP request when slow", t, func() {
		testSlow = true
		defer func() {
			testSlow = false
		}()
		client := NewDefaultHTTPClient()
		defer client.CloseIdleConnections()

		resp, err := client.Get(testServer.URL)
		gc.So(err, gc.ShouldBeNil)
		ioutil.ReadAll(resp.Body)
		gc.So(resp.Body.Close(), gc.ShouldBeNil)
		client.CloseIdleConnections()

		bytesRead, bytesWritten := client.BytesReadWritten()
		gc.So(bytesRead, gc.ShouldEqual, 134)
		gc.So(bytesWritten, gc.ShouldEqual, 96)
	})
}
