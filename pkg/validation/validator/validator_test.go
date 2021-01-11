/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hyperledger/fabric-protos-go/peer"
	sema "github.com/hyperledger/fabric/common/semaphore"
	validatorv20 "github.com/hyperledger/fabric/core/committer/txvalidator/v20"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	gcommon "github.com/hyperledger/fabric/gossip/common"
	viper "github.com/spf13/viper2015"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
	vmocks "github.com/trustbloc/fabric-peer-ext/pkg/validation/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/validation/validationresults"
)

const (
	org1MSPID = "Org1MSP"

	p1Org1Endpoint = "p1.org1.com"
	p2Org1Endpoint = "p2.org1.com"
	p3Org1Endpoint = "p3.org1.com"
)

// ensure roles are initialized
var _ = roles.GetRoles()

var (
	p1Org1PKIID = gcommon.PKIidType("pkiid_P1O1")
	p2Org1PKIID = gcommon.PKIidType("pkiid_P2O1")
	p3Org1PKIID = gcommon.PKIidType("pkiid_P3O1")
)

func TestProvider(t *testing.T) {
	providers := &Providers{
		Gossip: &mocks.GossipProvider{},
		Idp:    &mocks.IdentityDeserializerProvider{},
	}

	p := NewProvider(providers)
	require.NotNil(t, p)

	v := NewTxValidator(channelID, nil, &vmocks.ChannelResources{}, nil, nil, nil, nil, nil, nil)
	require.NotNil(t, v)

	require.Panics(t, func() {
		p.createValidator(channelID, nil, &vmocks.ChannelResources{}, nil, nil, nil, nil, nil, nil)
	})
}

func TestValidator_Validate(t *testing.T) {
	bb := mocks.NewBlockBuilder(channelID, 1000)
	bb.Transaction(txID1, peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction(txID2, peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction(txID3, peer.TxValidationCode_NOT_VALIDATED)

	block := bb.Build()

	ctOldVal := viper.Get(config.ConfValidationCommitterTransactionThreshold)
	viper.Set(config.ConfValidationCommitterTransactionThreshold, 1)

	sptOldVal := viper.Get(config.ConfValidationSinglePeerTransactionThreshold)
	viper.Set(config.ConfValidationSinglePeerTransactionThreshold, 1)

	defer func() {
		viper.Set(config.ConfValidationSinglePeerTransactionThreshold, sptOldVal)
		viper.Set(config.ConfValidationCommitterTransactionThreshold, ctOldVal)
	}()

	t.Run("Success", func(t *testing.T) {
		reset := roles.SetRole(roles.CommitterRole, roles.ValidatorRole)
		defer reset()

		v := createValidatorWithMocks(t, mocks.NewMockGossipAdapter().
			Self(org1MSPID, mocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
			Member(org1MSPID, mocks.NewMember(p2Org1Endpoint, p2Org1PKIID, roles.ValidatorRole)).
			Member(org1MSPID, mocks.NewMember(p3Org1Endpoint, p3Org1PKIID, roles.ValidatorRole)),
		)

		v.txValidator = vmocks.NewTxValidator().
			WithValidationResult(&validatorv20.BlockValidationResult{
				TIdx:           0,
				Txid:           txID1,
				ValidationCode: peer.TxValidationCode_VALID,
			}).
			WithValidationResult(&validatorv20.BlockValidationResult{
				TIdx:           1,
				Txid:           txID2,
				ValidationCode: peer.TxValidationCode_MVCC_READ_CONFLICT,
			}).
			WithValidationResult(&validatorv20.BlockValidationResult{
				TIdx:           2,
				Txid:           txID3,
				ValidationCode: peer.TxValidationCode_INVALID_ENDORSER_TRANSACTION,
			})

		err := v.Validate(block)
		require.NoError(t, err)

		r := newTxResults(channelID, block)
		require.Equal(t, peer.TxValidationCode_VALID, r.Flags().Flag(0))
		require.Equal(t, peer.TxValidationCode_MVCC_READ_CONFLICT, r.Flags().Flag(1))
		require.Equal(t, peer.TxValidationCode_INVALID_ENDORSER_TRANSACTION, r.Flags().Flag(2))
	})

	t.Run("No validators or committers warning", func(t *testing.T) {
		reset := roles.SetRole(roles.EndorserRole)
		defer reset()

		v := createValidatorWithMocks(t, mocks.NewMockGossipAdapter().
			Self(org1MSPID, mocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
			Member(org1MSPID, mocks.NewMember(p2Org1Endpoint, p2Org1PKIID, roles.EndorserRole)).
			Member(org1MSPID, mocks.NewMember(p3Org1Endpoint, p3Org1PKIID, roles.EndorserRole)),
		)

		v.txValidator = vmocks.NewTxValidator().
			WithValidationResult(&validatorv20.BlockValidationResult{
				TIdx:           0,
				Txid:           txID1,
				ValidationCode: peer.TxValidationCode_VALID,
			}).
			WithValidationResult(&validatorv20.BlockValidationResult{
				TIdx:           1,
				Txid:           txID2,
				ValidationCode: peer.TxValidationCode_VALID,
			}).
			WithValidationResult(&validatorv20.BlockValidationResult{
				TIdx:           2,
				Txid:           txID3,
				ValidationCode: peer.TxValidationCode_VALID,
			})

		err := v.Validate(block)
		require.NoError(t, err)
	})

	t.Run("Validation error", func(t *testing.T) {
		reset := roles.SetRole(roles.CommitterRole, roles.ValidatorRole)
		defer reset()

		gossip := mocks.NewMockGossipAdapter().
			Self(org1MSPID, mocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
			Member(org1MSPID, mocks.NewMember(p2Org1Endpoint, p2Org1PKIID, roles.ValidatorRole)).
			Member(org1MSPID, mocks.NewMember(p3Org1Endpoint, p3Org1PKIID, roles.ValidatorRole))

		// A remote peer returns a validation error so the local peer will re-validate and succeed
		t.Run("Remote error & local success", func(t *testing.T) {
			v := createValidatorWithMocks(t, gossip)

			v.txValidator = vmocks.NewTxValidator().
				WithValidationResult(&validatorv20.BlockValidationResult{
					TIdx: 0,
					Txid: txID1,
					Err:  fmt.Errorf("injected validation error"),
				}).
				WithValidationResult(&validatorv20.BlockValidationResult{
					TIdx:           1,
					Txid:           txID2,
					ValidationCode: peer.TxValidationCode_VALID,
				}).
				WithValidationResult(&validatorv20.BlockValidationResult{
					TIdx:           2,
					Txid:           txID3,
					ValidationCode: peer.TxValidationCode_VALID,
				}).
				WithValidationResult(&validatorv20.BlockValidationResult{
					TIdx:           0,
					Txid:           txID1,
					ValidationCode: peer.TxValidationCode_VALID,
				}).
				WithValidationResult(&validatorv20.BlockValidationResult{
					TIdx:           1,
					Txid:           txID2,
					ValidationCode: peer.TxValidationCode_VALID,
				}).
				WithValidationResult(&validatorv20.BlockValidationResult{
					TIdx:           2,
					Txid:           txID3,
					ValidationCode: peer.TxValidationCode_VALID,
				})

			err := v.Validate(block)
			require.NoError(t, err)
		})

		// A remote peer returns a validation error so the local peer will re-validate and fail
		t.Run("Remote error & local error", func(t *testing.T) {
			v := createValidatorWithMocks(t, gossip)

			v.txValidator = vmocks.NewTxValidator().
				WithValidationResult(&validatorv20.BlockValidationResult{
					TIdx:           0,
					Txid:           txID1,
					ValidationCode: peer.TxValidationCode_VALID,
				}).
				WithValidationResult(&validatorv20.BlockValidationResult{
					TIdx:           1,
					Txid:           txID2,
					ValidationCode: peer.TxValidationCode_VALID,
				}).
				WithValidationResult(&validatorv20.BlockValidationResult{
					TIdx: 2,
					Txid: txID3,
					Err:  fmt.Errorf("injected validation error"),
				}).
				WithValidationResult(&validatorv20.BlockValidationResult{
					TIdx:           0,
					Txid:           txID1,
					ValidationCode: peer.TxValidationCode_VALID,
				}).
				WithValidationResult(&validatorv20.BlockValidationResult{
					TIdx:           1,
					Txid:           txID2,
					ValidationCode: peer.TxValidationCode_VALID,
				}).
				WithValidationResult(&validatorv20.BlockValidationResult{
					TIdx: 2,
					Txid: txID3,
					Err:  fmt.Errorf("injected validation error"),
				})

			err := v.Validate(block)
			require.Error(t, err)
			require.Contains(t, err.Error(), "injected validation error")
		})

		t.Run("Validation cancelled", func(t *testing.T) {
			reset := roles.SetRole(roles.CommitterRole, roles.ValidatorRole)
			defer reset()

			v := createValidatorWithMocks(t, mocks.NewMockGossipAdapter().
				Self(org1MSPID, mocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
				Member(org1MSPID, mocks.NewMember(p2Org1Endpoint, p2Org1PKIID, roles.ValidatorRole)).
				Member(org1MSPID, mocks.NewMember(p3Org1Endpoint, p3Org1PKIID, roles.ValidatorRole)),
			)

			sem := &vmocks.Semaphore{}
			sem.AcquireReturns(context.Canceled)
			v.semaphore = sem

			_, _, _, err := v.validateBlock(context.Background(), block, func(txIdx int) bool {
				return true
			})
			require.EqualError(t, err, "context canceled")
		})
	})
}

func TestProvider_GetValidatorForChannel(t *testing.T) {
	const (
		channel1 = "channel1"
		channel2 = "channel2"
	)

	gossipProvider := &mocks.GossipProvider{}
	gossipProvider.GetGossipServiceReturns(mocks.NewMockGossipAdapter())

	providers := &Providers{
		Gossip: gossipProvider,
		Idp:    &mocks.IdentityDeserializerProvider{},
	}

	p := NewProvider(providers)
	require.NotNil(t, p)

	v1 := p.createValidator(channel1, &vmocks.Semaphore{}, &vmocks.ChannelResources{}, nil, nil, nil, nil, nil, nil)
	require.NotNil(t, v1)

	require.True(t, v1 == p.GetValidatorForChannel(channel1))
	require.Nil(t, p.GetValidatorForChannel(channel2))
}

func TestValidator_SubmitValidationResults(t *testing.T) {
	v := createValidatorWithMocks(t, mocks.NewMockGossipAdapter().
		Self(org1MSPID, mocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
		Member(org1MSPID, mocks.NewMember(p2Org1Endpoint, p2Org1PKIID, roles.ValidatorRole)).
		Member(org1MSPID, mocks.NewMember(p3Org1Endpoint, p3Org1PKIID, roles.ValidatorRole)),
	)

	results := &validationresults.Results{
		BlockNumber: 1000,
	}

	v.SubmitValidationResults(results)

	select {
	case r := <-v.resultsChan:
		require.True(t, r == results)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Did not receive submitted validation results")
	}
}

func TestValidator_ValidatePartial(t *testing.T) {
	v := createValidatorWithMocks(t, mocks.NewMockGossipAdapter().
		Self(org1MSPID, mocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
		Member(org1MSPID, mocks.NewMember(p2Org1Endpoint, p2Org1PKIID, roles.ValidatorRole)).
		Member(org1MSPID, mocks.NewMember(p3Org1Endpoint, p3Org1PKIID, roles.ValidatorRole)),
	)

	bb := mocks.NewBlockBuilder(channelID, 1000)
	bb.Transaction(txID1, peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction(txID2, peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction(txID3, peer.TxValidationCode_NOT_VALIDATED)

	block := bb.Build()

	t.Run("Success", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		flags, ids, err := v.ValidatePartial(ctx, block)
		require.NoError(t, err)
		require.NotEmpty(t, flags)
		require.Len(t, ids, len(flags))
	})

	t.Run("Canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		flags, ids, err := v.ValidatePartial(ctx, block)
		require.EqualError(t, err, context.Canceled.Error())
		require.Empty(t, flags)
		require.Empty(t, ids)
	})
}

func createValidatorWithMocks(t *testing.T, gossip gossipapi.GossipService) *validator {
	gossipProvider := &mocks.GossipProvider{}
	gossipProvider.GetGossipServiceReturns(gossip)

	providers := &Providers{
		Gossip: gossipProvider,
		Idp:    &mocks.IdentityDeserializerProvider{},
	}

	p := NewProvider(providers)
	require.NotNil(t, p)

	v := p.createValidator(channelID, sema.New(2), &vmocks.ChannelResources{}, nil, nil, nil, nil, nil, nil)
	require.NotNil(t, v)

	return v
}
