/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package policy

import (
	"strings"
	"testing"

	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTransientDataCollectionConfig(t *testing.T) {
	coll1 := "mycollection"

	var signers = [][]byte{[]byte("signer0"), []byte("signer1")}

	v := NewValidator()

	t.Run("No collection type -> success", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		collConfig := createStaticCollectionConfig(coll1, policyEnvelope, 1, 2, 1000)
		err := v.Validate(collConfig)
		assert.NoError(t, err)
	})

	t.Run("Private collection type -> success", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		collConfig := createPrivateCollectionConfig(coll1, policyEnvelope, 1, 2, 1000)
		err := v.Validate(collConfig)
		assert.NoError(t, err)
	})

	t.Run("Transient collection -> success", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		err := v.Validate(createTransientCollectionConfig(coll1, policyEnvelope, 1, 2, "1m"))
		assert.NoError(t, err)
	})
}

func TestValidateOffLedgerCollectionConfig(t *testing.T) {
	coll1 := "mycollection"

	var signers = [][]byte{[]byte("signer0"), []byte("signer1")}

	v := NewValidator()

	t.Run("Off-Ledger collection -> success", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		err := v.Validate(createOffLedgerCollectionConfig(coll1, policyEnvelope, 1, 2, "1m"))
		assert.NoError(t, err)
	})

	t.Run("Off-Ledger req == 0 -> error", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		err := v.Validate(createOffLedgerCollectionConfig(coll1, policyEnvelope, 0, 2, "1m"))
		require.Error(t, err)
		expectedErr := "required peer count must be greater than 0"
		assert.Truef(t, strings.Contains(err.Error(), expectedErr), "Expected error to contain '%s' but got '%s'", expectedErr, err)
	})

	t.Run("transient collection req > max -> error", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		err := v.Validate(createTransientCollectionConfig(coll1, policyEnvelope, 3, 2, "1m"))
		require.Error(t, err)
		expectedErr := "maximum peer count (2) must be greater than or equal to required peer count (3)"
		assert.Truef(t, strings.Contains(err.Error(), expectedErr), "Expected error to contain '%s' but got '%s'", expectedErr, err)
	})

	t.Run("Off-Ledger no time-to-live -> success", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		err := v.Validate(createOffLedgerCollectionConfig(coll1, policyEnvelope, 1, 2, ""))
		require.NoError(t, err)
	})

	t.Run("Off-Ledger invalid time-to-live -> error", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		err := v.Validate(createOffLedgerCollectionConfig(coll1, policyEnvelope, 1, 2, "1k"))
		require.Error(t, err)
		expectedErr := "invalid time format for time to live"
		assert.Truef(t, strings.Contains(err.Error(), expectedErr), "Expected error to contain '%s' but got '%s'", expectedErr, err)
	})

	t.Run("Off-Ledger with blocks-to-live -> error", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		config := createTransientCollectionConfig(coll1, policyEnvelope, 1, 2, "1m")
		config.GetStaticCollectionConfig().BlockToLive = 100
		err := v.Validate(config)
		require.Error(t, err)
		expectedErr := "block-to-live not supported"
		assert.Truef(t, strings.Contains(err.Error(), expectedErr), "Expected error to contain '%s' but got '%s'", expectedErr, err)
	})
}

func TestValidateDCASCollectionConfig(t *testing.T) {
	coll1 := "mycollection"

	var signers = [][]byte{[]byte("signer0"), []byte("signer1")}

	v := NewValidator()

	t.Run("DCAS collection -> success", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		err := v.Validate(createDCASCollectionConfig(coll1, policyEnvelope, 1, 2, "1m"))
		assert.NoError(t, err)
	})

	t.Run("DCAS req == 0 -> error", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		err := v.Validate(createDCASCollectionConfig(coll1, policyEnvelope, 0, 2, "1m"))
		require.Error(t, err)
		expectedErr := "required peer count must be greater than 0"
		assert.Truef(t, strings.Contains(err.Error(), expectedErr), "Expected error to contain '%s' but got '%s'", expectedErr, err)
	})

	t.Run("transient collection req > max -> error", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		err := v.Validate(createTransientCollectionConfig(coll1, policyEnvelope, 3, 2, "1m"))
		require.Error(t, err)
		expectedErr := "maximum peer count (2) must be greater than or equal to required peer count (3)"
		assert.Truef(t, strings.Contains(err.Error(), expectedErr), "Expected error to contain '%s' but got '%s'", expectedErr, err)
	})

	t.Run("DCAS no time-to-live -> success", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		err := v.Validate(createDCASCollectionConfig(coll1, policyEnvelope, 1, 2, ""))
		require.NoError(t, err)
	})

	t.Run("DCAS invalid time-to-live -> error", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		err := v.Validate(createDCASCollectionConfig(coll1, policyEnvelope, 1, 2, "1k"))
		require.Error(t, err)
		expectedErr := "invalid time format for time to live"
		assert.Truef(t, strings.Contains(err.Error(), expectedErr), "Expected error to contain '%s' but got '%s'", expectedErr, err)
	})

	t.Run("DCAS with blocks-to-live -> error", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		config := createTransientCollectionConfig(coll1, policyEnvelope, 1, 2, "1m")
		config.GetStaticCollectionConfig().BlockToLive = 100
		err := v.Validate(config)
		require.Error(t, err)
		expectedErr := "block-to-live not supported"
		assert.Truef(t, strings.Contains(err.Error(), expectedErr), "Expected error to contain '%s' but got '%s'", expectedErr, err)
	})
}

func TestValidateNewCollectionConfigAgainstOld(t *testing.T) {
	coll1 := "mycollection"

	var signers = [][]byte{[]byte("signer0"), []byte("signer1")}

	v := NewValidator()

	t.Run("updated -> success", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		oldCollConfig := createTransientCollectionConfig(coll1, policyEnvelope, 1, 2, "10m")
		newCollConfig := createTransientCollectionConfig(coll1, policyEnvelope, 2, 3, "20m")
		err := v.ValidateNewCollectionConfigsAgainstOld([]*common.CollectionConfig{newCollConfig}, []*common.CollectionConfig{oldCollConfig})
		assert.NoError(t, err)
	})

	t.Run("private collection updated to transient -> error", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		oldCollConfig := createStaticCollectionConfig(coll1, policyEnvelope, 1, 2, 1000)
		newCollConfig := createTransientCollectionConfig(coll1, policyEnvelope, 1, 2, "10m")
		err := v.ValidateNewCollectionConfigsAgainstOld([]*common.CollectionConfig{newCollConfig}, []*common.CollectionConfig{oldCollConfig})
		assert.EqualError(t, err, "collection-name: mycollection -- attempt to change collection type from [COL_PRIVATE] to [COL_TRANSIENT]")
	})

	t.Run("private collection updated to off-ledger -> error", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		oldCollConfig := createStaticCollectionConfig(coll1, policyEnvelope, 1, 2, 1000)
		newCollConfig := createOffLedgerCollectionConfig(coll1, policyEnvelope, 1, 2, "10m")
		err := v.ValidateNewCollectionConfigsAgainstOld([]*common.CollectionConfig{newCollConfig}, []*common.CollectionConfig{oldCollConfig})
		assert.EqualError(t, err, "collection-name: mycollection -- attempt to change collection type from [COL_PRIVATE] to [COL_OFFLEDGER]")
	})
}

func createTransientCollectionConfig(collectionName string, signaturePolicyEnvelope *common.SignaturePolicyEnvelope,
	requiredPeerCount int32, maximumPeerCount int32, ttl string) *common.CollectionConfig {
	signaturePolicy := &common.CollectionPolicyConfig_SignaturePolicy{
		SignaturePolicy: signaturePolicyEnvelope,
	}

	return &common.CollectionConfig{
		Payload: &common.CollectionConfig_StaticCollectionConfig{
			StaticCollectionConfig: &common.StaticCollectionConfig{
				Name:              collectionName,
				Type:              common.CollectionType_COL_TRANSIENT,
				MemberOrgsPolicy:  &common.CollectionPolicyConfig{Payload: signaturePolicy},
				RequiredPeerCount: requiredPeerCount,
				MaximumPeerCount:  maximumPeerCount,
				TimeToLive:        ttl,
			},
		},
	}
}

func createPrivateCollectionConfig(collectionName string, signaturePolicyEnvelope *common.SignaturePolicyEnvelope,
	requiredPeerCount int32, maximumPeerCount int32, blockToLive uint64) *common.CollectionConfig {
	config := createStaticCollectionConfig(collectionName, signaturePolicyEnvelope, requiredPeerCount, maximumPeerCount, blockToLive)
	config.GetStaticCollectionConfig().Type = common.CollectionType_COL_PRIVATE
	return config
}

func createStaticCollectionConfig(collectionName string, signaturePolicyEnvelope *common.SignaturePolicyEnvelope,
	requiredPeerCount int32, maximumPeerCount int32, blockToLive uint64) *common.CollectionConfig {
	signaturePolicy := &common.CollectionPolicyConfig_SignaturePolicy{
		SignaturePolicy: signaturePolicyEnvelope,
	}

	return &common.CollectionConfig{
		Payload: &common.CollectionConfig_StaticCollectionConfig{
			StaticCollectionConfig: &common.StaticCollectionConfig{
				Name:              collectionName,
				MemberOrgsPolicy:  &common.CollectionPolicyConfig{Payload: signaturePolicy},
				RequiredPeerCount: requiredPeerCount,
				MaximumPeerCount:  maximumPeerCount,
				BlockToLive:       blockToLive,
			},
		},
	}
}

func createOffLedgerCollectionConfig(collectionName string, signaturePolicyEnvelope *common.SignaturePolicyEnvelope,
	requiredPeerCount int32, maximumPeerCount int32, ttl string) *common.CollectionConfig {
	signaturePolicy := &common.CollectionPolicyConfig_SignaturePolicy{
		SignaturePolicy: signaturePolicyEnvelope,
	}

	return &common.CollectionConfig{
		Payload: &common.CollectionConfig_StaticCollectionConfig{
			StaticCollectionConfig: &common.StaticCollectionConfig{
				Name:              collectionName,
				Type:              common.CollectionType_COL_OFFLEDGER,
				MemberOrgsPolicy:  &common.CollectionPolicyConfig{Payload: signaturePolicy},
				RequiredPeerCount: requiredPeerCount,
				MaximumPeerCount:  maximumPeerCount,
				TimeToLive:        ttl,
			},
		},
	}
}

func createDCASCollectionConfig(collectionName string, signaturePolicyEnvelope *common.SignaturePolicyEnvelope,
	requiredPeerCount int32, maximumPeerCount int32, ttl string) *common.CollectionConfig {
	signaturePolicy := &common.CollectionPolicyConfig_SignaturePolicy{
		SignaturePolicy: signaturePolicyEnvelope,
	}

	return &common.CollectionConfig{
		Payload: &common.CollectionConfig_StaticCollectionConfig{
			StaticCollectionConfig: &common.StaticCollectionConfig{
				Name:              collectionName,
				Type:              common.CollectionType_COL_DCAS,
				MemberOrgsPolicy:  &common.CollectionPolicyConfig{Payload: signaturePolicy},
				RequiredPeerCount: requiredPeerCount,
				MaximumPeerCount:  maximumPeerCount,
				TimeToLive:        ttl,
			},
		},
	}
}
