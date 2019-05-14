/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package policy

import (
	"testing"

	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
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
