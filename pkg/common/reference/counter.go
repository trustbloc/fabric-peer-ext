/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package reference

import (
	"errors"
	"sync/atomic"
)

const (
	statusReady        = 0
	statusClosePending = 1
	statusClosed       = 2
)

// Counter counts the references to a resource. When the counter is closed then the
// provided close func is invoked only after the count reaches 0.
type Counter struct {
	count  int32
	closed uint32
	close  func()
}

// NewCounter returns a new Counter. The provided function is invoked when the
// counter is closed and the reference count is 0.
func NewCounter(closer func()) *Counter {
	return &Counter{
		close: closer,
	}
}

// Increment increments the count by 1. An error is returned if
// the function is invoked on a closed counter.
func (r *Counter) Increment() (int32, error) {
	if atomic.LoadUint32(&r.closed) != statusReady {
		return 0, errors.New("attempt to increment count on closed resource")
	}

	return atomic.AddInt32(&r.count, 1), nil
}

// Decrement decrements the count by 1. If the counter has already been
// closed AND the count reaches 0 then the close function is invoked.
// An error is returned if the function is invoked when the count is 0.
func (r *Counter) Decrement() (int32, error) {
	if atomic.LoadInt32(&r.count) == 0 {
		return 0, errors.New("attempt to decrement count when count is already 0")
	}

	newCount := atomic.AddInt32(&r.count, -1)
	if newCount == 0 && atomic.LoadUint32(&r.closed) == statusClosePending {
		r.notifyClosed()
	}

	return newCount, nil
}

// Close closes the counter, which will cause the close function
// to be invoked after the count reaches 0.
func (r *Counter) Close() {
	if !atomic.CompareAndSwapUint32(&r.closed, statusReady, statusClosePending) {
		// Already closed
		return
	}

	if atomic.LoadInt32(&r.count) == 0 {
		r.notifyClosed()
	}
}

func (r *Counter) notifyClosed() {
	if atomic.CompareAndSwapUint32(&r.closed, statusClosePending, statusClosed) {
		r.close()
	}
}
