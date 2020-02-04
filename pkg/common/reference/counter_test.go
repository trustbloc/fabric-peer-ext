/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package reference

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReference(t *testing.T) {
	closed := 0
	onClosed := func() {
		closed++
	}

	r := NewCounter(onClosed)
	require.NotNil(t, r)

	count, err := r.Increment()
	require.NoError(t, err)
	require.Equal(t, int32(1), count)

	count, err = r.Decrement()
	require.NoError(t, err)
	require.Equal(t, int32(0), count)

	require.Equal(t, 0, closed)
	r.Close()
	require.Equal(t, 1, closed)

	r.Close()
	require.Equal(t, 1, closed)

	count, err = r.Increment()
	require.EqualError(t, err, "attempt to increment count on closed resource")

	count, err = r.Decrement()
	require.EqualError(t, err, "attempt to decrement count when count is already 0")
}

func TestReferenceCloseOnLastRef(t *testing.T) {
	closed := false
	onClosed := func() {
		closed = true
	}

	r := NewCounter(onClosed)
	require.NotNil(t, r)

	count, err := r.Increment()
	require.Equal(t, int32(1), count)

	require.False(t, closed)
	r.Close()
	require.False(t, closed)

	count, err = r.Decrement()
	require.NoError(t, err)
	require.Equal(t, int32(0), count)
	require.True(t, closed)
}

func TestReference_Concurrent(t *testing.T) {
	var closed uint32
	onClosed := func() {
		atomic.AddUint32(&closed, 1)
	}

	r := NewCounter(onClosed)
	require.NotNil(t, r)

	n := 1000
	h := n / 2

	for i := 0; i < h; i++ {
		_, err := r.Increment()
		require.NoError(t, err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		for i := 0; i < h; i++ {
			_, err := r.Increment()
			require.NoError(t, err)
		}

		r.Close()
		require.Equal(t, uint32(0), atomic.LoadUint32(&closed))
		wg.Done()
	}()

	go func() {
		for i := 0; i < h; i++ {
			_, err := r.Decrement()
			require.NoError(t, err)
		}
		wg.Done()
	}()

	require.Equal(t, uint32(0), atomic.LoadUint32(&closed))

	wg.Wait()

	for i := 0; i < h; i++ {
		_, err := r.Decrement()
		require.NoError(t, err)
	}

	require.Equal(t, uint32(1), atomic.LoadUint32(&closed))
}
