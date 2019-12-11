/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cdbblkstorage

import (
	"testing"

	"github.com/trustbloc/fabric-peer-ext/pkg/common/util"
	"github.com/trustbloc/fabric-peer-ext/pkg/testutil"

	"github.com/hyperledger/fabric/protoutil"

	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	"github.com/stretchr/testify/assert"
)

type testEnv struct {
	t        testing.TB
	provider *CDBBlockstoreProvider
}

func newTestEnv(t testing.TB) *testEnv {
	attrsToIndex := []blkstorage.IndexableAttr{
		blkstorage.IndexableAttrBlockHash,
		blkstorage.IndexableAttrBlockNum,
		blkstorage.IndexableAttrTxID,
		blkstorage.IndexableAttrBlockNumTranNum,
		//blkstorage.IndexableAttrBlockTxID,
		//blkstorage.IndexableAttrTxValidationCode,
	}
	env, err := newTestEnvSelectiveIndexing(t, attrsToIndex)
	assert.NoError(t, err)
	return env
}

func newTestEnvSelectiveIndexing(t testing.TB, attrsToIndex []blkstorage.IndexableAttr) (*testEnv, error) {
	indexConfig := &blkstorage.IndexConfig{AttrsToIndex: attrsToIndex}
	privider, err := NewProvider(indexConfig, testutil.TestLedgerConf())
	if err != nil {
		return nil, err
	}
	return &testEnv{t, privider.(*CDBBlockstoreProvider)}, nil
}

func (env *testEnv) Cleanup() {
	env.provider.Close()
}

func extractTxID(txEnvelopBytes []byte) (string, error) {
	txEnvelope, err := protoutil.GetEnvelopeFromBlock(txEnvelopBytes)
	if err != nil {
		return "", err
	}
	txPayload, err := util.ExtractPayload(txEnvelope)
	if err != nil {
		return "", nil
	}
	chdr, err := protoutil.UnmarshalChannelHeader(txPayload.Header.ChannelHeader)
	if err != nil {
		return "", err
	}
	return chdr.TxId, nil
}
