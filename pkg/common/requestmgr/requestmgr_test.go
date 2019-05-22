/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package requestmgr

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	channel1 = "channel1"
	channel2 = "channel2"

	timeout = 200 * time.Millisecond
)

func TestElements_Get(t *testing.T) {
	const (
		ns1   = "ns1"
		coll1 = "coll1"
		coll2 = "coll2"
		key1  = "key1"
		key2  = "key2"
		key3  = "key3"
	)

	e1 := &Element{Namespace: ns1, Collection: coll1, Key: key1, Expiry: time.Now().Add(time.Minute)}
	e2 := &Element{Namespace: ns1, Collection: coll2, Key: key1, Expiry: time.Now().Add(time.Minute)}
	e3 := &Element{Namespace: ns1, Collection: coll1, Key: key2, Expiry: time.Now().Add(time.Minute)}

	elements := Elements{e1, e2, e3}

	e, ok := elements.Get(ns1, coll1, key1)
	assert.True(t, ok)
	require.NotNil(t, e)
	assert.Equal(t, e1, e)

	e, ok = elements.Get(ns1, coll2, key1)
	assert.True(t, ok)
	require.NotNil(t, e)
	assert.Equal(t, e2, e)

	e, ok = elements.Get(ns1, coll1, key3)
	assert.False(t, ok)
	require.Nil(t, e)
}

func TestResponseMgr(t *testing.T) {
	t.Parallel()

	m1 := Get(channel1)
	require.NotNil(t, m1)

	m2 := Get(channel2)
	require.NotNil(t, m2)
	require.False(t, m1 == m2)

	t.Run("Response received - channel1", func(t *testing.T) {
		t.Parallel()

		req1 := m1.NewRequest()
		require.NotNil(t, req1)

		req2 := m1.NewRequest()
		require.NotNil(t, req2)

		expect1 := &Response{Endpoint: "p1"}
		expect2 := &Response{Endpoint: "p2"}

		go func() {
			m1.Respond(req1.ID(), expect1)
			m1.Respond(req2.ID(), expect2)
		}()

		res1, err := req1.GetResponse(context.Background())
		require.NoError(t, err)
		assert.Equal(t, expect1, res1)

		res2, err := req2.GetResponse(context.Background())
		require.NoError(t, err)
		assert.Equal(t, expect2, res2)
	})

	t.Run("Response received - channel2", func(t *testing.T) {
		t.Parallel()

		req3 := m2.NewRequest()
		require.NotNil(t, req3)

		expect3 := &Response{Endpoint: "p3"}

		go func() {
			m2.Respond(req3.ID(), expect3)
		}()

		res3, err := req3.GetResponse(context.Background())
		require.NoError(t, err)
		assert.Equal(t, expect3, res3)
	})

	t.Run("No response", func(t *testing.T) {
		t.Parallel()

		req4 := m2.NewRequest()
		require.NotNil(t, req4)

		// Should time out since we never publish a response
		ctxt, _ := context.WithTimeout(context.Background(), timeout)
		_, err := req4.GetResponse(ctxt)
		require.Error(t, err)
	})

	t.Run("Context cancelled", func(t *testing.T) {
		t.Parallel()

		req5 := m2.NewRequest()
		require.NotNil(t, req5)

		ctxt, cancel := context.WithTimeout(context.Background(), timeout)

		go func() {
			cancel()
			time.Sleep(10 * time.Millisecond)
			m2.Respond(req5.ID(), &Response{}) // No subscribers since the context was cancelled
		}()

		// Should get error that the context was cancelled
		_, err := req5.GetResponse(ctxt)
		require.Error(t, err)
		assert.EqualError(t, err, "context canceled")
	})

	t.Run("Request cancelled", func(t *testing.T) {
		t.Parallel()

		req := m2.NewRequest()
		require.NotNil(t, req)
		req.Cancel()

		// Should get error that the request was cancelled
		_, err := req.GetResponse(context.Background())
		require.Error(t, err)
		assert.EqualError(t, err, "request has already completed")
	})
}
