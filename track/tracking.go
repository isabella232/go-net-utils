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

// ByteTracker represents a type that is capable
// of tracking bytes read and written over a period of time
type ByteTracker interface {

	// BytesReadWritten should only ever be called when the caller knows
	// that no more mutations to the bytes read or written happens past the
	// point of calling. In the context of an HTTP request and its associated
	// socket. It's possible that a connection with the same ByteTracker may be
	// reused. If this is the case, the caller is responsible for calling
	// ResetBytes.
	BytesReadWritten() (read uint64, written uint64)
	ResetBytes()
}
