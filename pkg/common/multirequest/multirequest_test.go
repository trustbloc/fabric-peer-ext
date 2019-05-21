/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multirequest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/common"
)

type requestTest struct {
	id     string
	err    error
	values [][]byte
}

func TestAllSet(t *testing.T) {
	t.Parallel()

	re := New()

	value1 := []byte("value1")
	value2 := []byte("value2")
	value3 := []byte("value3")

	requests := []requestTest{
		{id: "Request1", values: [][]byte{value1, value2, value3}},
		{id: "Request2", values: [][]byte{value1, value2, value3}},
		{id: "Request3", values: [][]byte{value1, value2, value3}},
	}

	for _, r := range requests {
		re.Add(r.id, getRequestFunc(r))
	}

	result := re.Execute(context.Background())
	require.NotNil(t, result)
	require.Equal(t, 3, len(result.Values))
	assert.Equal(t, value1, result.Values[0])
	assert.Equal(t, value2, result.Values[1])
	assert.Equal(t, value3, result.Values[2])
}

func TestMerge(t *testing.T) {
	t.Parallel()

	re := New()

	value1 := []byte("value1")
	value2 := []byte("value2")
	value3 := []byte("value3")

	requests := []requestTest{
		{id: "Request1", values: [][]byte{nil, value2, value3}},
		{id: "Request2", values: [][]byte{value1, nil, value3}},
		{id: "Request3", values: [][]byte{value1, value2, nil}},
	}

	for _, r := range requests {
		re.Add(r.id, getRequestFunc(r))
	}

	result := re.Execute(context.Background())
	require.NotNil(t, result)
	require.Equal(t, 3, len(result.Values))

	assert.Equal(t, value1, result.Values[0])
	assert.Equal(t, value2, result.Values[1])
	assert.Equal(t, value3, result.Values[2])
}

func TestTimeoutOrCancel(t *testing.T) {
	t.Parallel()

	re := New()

	requests := []requestTest{
		{id: "Request1"},
		{id: "Request2"},
		{id: "Request3"},
	}

	for _, r := range requests {
		re.Add(r.id, getRequestFunc(r))
	}

	t.Run("Timeout", func(t *testing.T) {
		t.Parallel()
		ctxt, _ := context.WithTimeout(context.Background(), 1*time.Microsecond)
		result := re.Execute(ctxt)
		assert.True(t, result.Values.IsEmpty())
	})

	t.Run("Cancelled", func(t *testing.T) {
		t.Parallel()
		ctxt, cancel := context.WithCancel(context.Background())
		go cancel()
		result := re.Execute(ctxt)
		assert.True(t, result.Values.IsEmpty())
	})
}

func TestSomeFailed(t *testing.T) {
	t.Parallel()

	re := New()

	value1 := []byte("value1")
	value3 := []byte("value3")

	requests := []requestTest{
		{id: "Request1", err: errors.New("error for request1")},
		{id: "Request2", values: [][]byte{value1, nil, value3}},
		{id: "Request3", err: errors.New("error for request3")},
	}

	for _, r := range requests {
		re.Add(r.id, getRequestFunc(r))
	}

	result := re.Execute(context.Background())
	require.Equal(t, 3, len(result.Values))
	assert.Equal(t, value1, result.Values[0])
	assert.Nil(t, result.Values[1])
	assert.Equal(t, value3, result.Values[2])
}

func getRequestFunc(r requestTest) Request {
	return func(ctxt context.Context) (common.Values, error) {
		select {
		case <-time.After(5 * time.Millisecond):
			return asValues(r.values), r.err
		case <-ctxt.Done():
			return nil, ctxt.Err()
		}
	}
}

func asValues(bv [][]byte) common.Values {
	vals := make(common.Values, len(bv))
	for i, v := range bv {
		vals[i] = v
	}
	return vals
}
