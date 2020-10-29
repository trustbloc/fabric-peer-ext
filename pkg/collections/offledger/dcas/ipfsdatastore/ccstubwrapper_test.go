/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ipfsdatastore

import (
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/ipfs/go-datastore"
	"github.com/stretchr/testify/require"
)

//go:generate counterfeiter -o ../../mocks/offledgerclient.gen.go --fake-name OffLedgerClient ../../../client OffLedger

func TestStubWrapper_Put(t *testing.T) {
	value1 := []byte("value1")

	c := NewStubWrapper(coll1, shimtest.NewMockStub(ns1, &mockChaincode{}))
	require.NotNil(t, c)
	require.NoError(t, c.Put(datastore.NewKey(key1), value1))
}

func TestStubWrapper_Get(t *testing.T) {
	t.Run("Get - not found -> success", func(t *testing.T) {
		c := NewStubWrapper(coll1, shimtest.NewMockStub(ns1, &mockChaincode{}))
		require.NotNil(t, c)
		value, err := c.Get(datastore.NewKey(key1))
		require.EqualError(t, err, datastore.ErrNotFound.Error())
		require.Empty(t, value)
	})

	t.Run("Get -> success", func(t *testing.T) {
		value1 := []byte("value1")

		dskey := datastore.NewKey(key1)

		stub := shimtest.NewMockStub(ns1, &mockChaincode{})
		require.NoError(t, stub.PutPrivateData(coll1, dskey.String(), value1))

		c := NewStubWrapper(coll1, stub)
		require.NotNil(t, c)

		value, err := c.Get(dskey)
		require.NoError(t, err)
		require.Equal(t, value1, value)
	})
}

type mockChaincode struct {
}

func (cc *mockChaincode) Init(shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}
func (cc *mockChaincode) Invoke(shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}
