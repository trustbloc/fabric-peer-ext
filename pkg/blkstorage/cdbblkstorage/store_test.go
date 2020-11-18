/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cdbblkstorage

import (
	"fmt"
	"testing"

	"github.com/hyperledger/fabric-protos-go/peer"
	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/blkstorage/cdbblkstorage/mocks"
)

func TestStore_RetrieveBlockByNumber(t *testing.T) {
	t.Run("DB error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected DB error")

		blockStore := &mocks.CouchDB{}
		blockStore.ReadDocReturns(nil, "", errExpected)

		s := newStore(channel1, blockStore, &mocks.CouchDB{})
		require.NotNil(t, s)

		b, err := s.RetrieveBlockByNumber(1000)
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.Nil(t, b)
	})

	t.Run("Unmarshal error", func(t *testing.T) {
		blockStore := &mocks.CouchDB{}

		doc := &couchdb.CouchDoc{
			Attachments: []*couchdb.AttachmentInfo{
				{
					Name:            blockAttachmentName,
					AttachmentBytes: []byte("invalid attachment"),
				},
			},
		}

		blockStore.ReadDocReturns(doc, "", nil)

		s := newStore(channel1, blockStore, &mocks.CouchDB{})
		require.NotNil(t, s)

		b, err := s.RetrieveBlockByNumber(1000)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unmarshal of block [1000] from couchDB failed")
		require.Nil(t, b)
	})
}

func TestStore_RetrieveBlockByTxID(t *testing.T) {
	t.Run("DB error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected DB error")

		txnStore := &mocks.CouchDB{}
		txnStore.ReadDocReturns(nil, "", errExpected)

		s := newStore(channel1, &mocks.CouchDB{}, txnStore)
		require.NotNil(t, s)

		b, err := s.RetrieveBlockByTxID("tx1")
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.Nil(t, b)
	})

	t.Run("Unmarshal error", func(t *testing.T) {
		txnStore := &mocks.CouchDB{}

		txnStore.ReadDocReturns(&couchdb.CouchDoc{}, "", nil)

		s := newStore(channel1, &mocks.CouchDB{}, txnStore)
		require.NotNil(t, s)

		b, err := s.RetrieveBlockByTxID("tx1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "result from DB is not JSON encoded")
		require.Nil(t, b)
	})

	t.Run("Block hash not found", func(t *testing.T) {
		txnStore := &mocks.CouchDB{}

		doc := &couchdb.CouchDoc{
			JSONValue: []byte(`{"block_number":0}`),
		}

		txnStore.ReadDocReturns(doc, "", nil)

		s := newStore(channel1, &mocks.CouchDB{}, txnStore)
		require.NotNil(t, s)

		b, err := s.RetrieveBlockByTxID("tx1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "block hash was not found for transaction ID")
		require.Nil(t, b)
	})

	t.Run("Invalid block hash", func(t *testing.T) {
		txnStore := &mocks.CouchDB{}

		doc := &couchdb.CouchDoc{
			JSONValue: []byte(`{"block_number":0,"block_hash":-1}`),
		}

		txnStore.ReadDocReturns(doc, "", nil)

		s := newStore(channel1, &mocks.CouchDB{}, txnStore)
		require.NotNil(t, s)

		b, err := s.RetrieveBlockByTxID("tx1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "block hash has invalid type for transaction ID")
		require.Nil(t, b)
	})

	t.Run("Decode error", func(t *testing.T) {
		txnStore := &mocks.CouchDB{}

		doc := &couchdb.CouchDoc{
			JSONValue: []byte(`{"block_number":0,"block_hash":"xxx"}`),
		}

		txnStore.ReadDocReturns(doc, "", nil)

		s := newStore(channel1, &mocks.CouchDB{}, txnStore)
		require.NotNil(t, s)

		b, err := s.RetrieveBlockByTxID("tx1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "block hash was invalid for transaction ID")
		require.Nil(t, b)
	})
}

func TestStore_RetrieveTxByID(t *testing.T) {
	t.Run("DB error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected DB error")

		txnStore := &mocks.CouchDB{}
		txnStore.ReadDocReturns(nil, "", errExpected)

		s := newStore(channel1, &mocks.CouchDB{}, txnStore)
		require.NotNil(t, s)

		envelope, err := s.RetrieveTxByID("tx1")
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.Nil(t, envelope)
	})

	t.Run("Unmarshal error", func(t *testing.T) {
		txnStore := &mocks.CouchDB{}

		doc := &couchdb.CouchDoc{
			Attachments: []*couchdb.AttachmentInfo{
				{
					Name:            blockAttachmentName,
					AttachmentBytes: []byte("invalid attachment"),
				},
			},
		}

		txnStore.ReadDocReturns(doc, "", nil)

		s := newStore(channel1, &mocks.CouchDB{}, txnStore)
		require.NotNil(t, s)

		envelope, err := s.RetrieveTxByID("tx1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "result from DB is not JSON encoded")
		require.Nil(t, envelope)
	})
}

func TestStore_RetrieveTxValidationCodeByTxID(t *testing.T) {
	t.Run("DB error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected DB error")

		txnStore := &mocks.CouchDB{}
		txnStore.ReadDocReturns(nil, "", errExpected)

		s := newStore(channel1, &mocks.CouchDB{}, txnStore)
		require.NotNil(t, s)

		code, err := s.RetrieveTxValidationCodeByTxID("tx1")
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.Equal(t, peer.TxValidationCode(-1), code)
	})

	t.Run("Unmarshal error", func(t *testing.T) {
		txnStore := &mocks.CouchDB{}

		txnStore.ReadDocReturns(&couchdb.CouchDoc{}, "", nil)

		s := newStore(channel1, &mocks.CouchDB{}, txnStore)
		require.NotNil(t, s)

		code, err := s.RetrieveTxValidationCodeByTxID("tx1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "result from DB is not JSON encoded")
		require.Equal(t, peer.TxValidationCode(-1), code)
	})

	t.Run("Validation code not found", func(t *testing.T) {
		txnStore := &mocks.CouchDB{}

		doc := &couchdb.CouchDoc{
			JSONValue: []byte(`{"block_number":0}`),
		}

		txnStore.ReadDocReturns(doc, "", nil)

		s := newStore(channel1, &mocks.CouchDB{}, txnStore)
		require.NotNil(t, s)

		code, err := s.RetrieveTxValidationCodeByTxID("tx1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "validation code was not found for transaction ID")
		require.Equal(t, peer.TxValidationCode(255), code)
	})

	t.Run("Invalid validation code", func(t *testing.T) {
		txnStore := &mocks.CouchDB{}

		doc := &couchdb.CouchDoc{
			JSONValue: []byte(`{"block_number":0,"validation_code":-1}`),
		}

		txnStore.ReadDocReturns(doc, "", nil)

		s := newStore(channel1, &mocks.CouchDB{}, txnStore)
		require.NotNil(t, s)

		code, err := s.RetrieveTxValidationCodeByTxID("tx1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "validation code has invalid type for transaction ID")
		require.Equal(t, peer.TxValidationCode(255), code)
	})

	t.Run("Parse error", func(t *testing.T) {
		txnStore := &mocks.CouchDB{}

		doc := &couchdb.CouchDoc{
			JSONValue: []byte(`{"block_number":0,"validation_code":"xxx"}`),
		}

		txnStore.ReadDocReturns(doc, "", nil)

		s := newStore(channel1, &mocks.CouchDB{}, txnStore)
		require.NotNil(t, s)

		code, err := s.RetrieveTxValidationCodeByTxID("tx1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "validation code was invalid for transaction ID")
		require.Equal(t, peer.TxValidationCode(255), code)
	})
}
