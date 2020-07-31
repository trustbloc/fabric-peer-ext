/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package proprespvalidator

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	txnmocks "github.com/trustbloc/fabric-peer-ext/pkg/txn/mocks"
)

const (
	channel1ID = "channel1"
	channel2ID = "channel2"
)

func TestValidatorMgr_ValidatorForChannel(t *testing.T) {
	lp := &mocks.LedgerProvider{}
	idd := &mocks.IdentityDeserializerProvider{}
	bpp := mocks.NewBlockPublisherProvider()
	cci := &txnmocks.LifecycleCCInfoProvider{}

	p := New(lp, idd, bpp, cci)
	require.NotNil(t, p)

	v := p.ValidatorForChannel(channel1ID)
	require.NotNil(t, v)

	v2 := p.ValidatorForChannel(channel1ID)
	require.True(t, v == v2)

	v3 := p.ValidatorForChannel(channel2ID)
	require.False(t, v == v3)
}
