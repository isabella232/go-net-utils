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
	"net/http"
	"time"
)

// HTTPRoundTripper wraps an http.RoundTripper and tracks reads and writes
type HTTPRoundTripper interface {
	http.RoundTripper
	ByteTracker
}

type basicHTTPHTTPRoundTripper struct {
	http.RoundTripper
	Dialer
}

// NewHTTPRoundTripper returns a new HTTPRoundTripper wrapping
// the given http.Transport and net.Dialer
func NewHTTPRoundTripper(
	innerTransport *http.Transport,
	innerDialer *net.Dialer,
) HTTPRoundTripper {
	dialer := &basicDialer{
		Dialer: innerDialer,
	}
	innerTransport.DialContext = dialer.DialContext

	return &basicHTTPHTTPRoundTripper{
		RoundTripper: innerTransport,
		Dialer:       dialer,
	}
}

// NewDefaultHTTPRoundTripper returns a new HTTPRoundTripper based
// on a default net.Dialer and http.Transport
func NewDefaultHTTPRoundTripper() HTTPRoundTripper {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}
	innerTransport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		IdleConnTimeout:       1 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return NewHTTPRoundTripper(innerTransport, dialer)
}
