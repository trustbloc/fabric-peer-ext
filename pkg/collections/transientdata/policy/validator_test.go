/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package policy

import (
	"strings"
	"testing"

	"github.com/hyperledger/fabric-protos-go/common"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/policydsl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateConfig(t *testing.T) {
	coll1 := "mycollection"

	var signers = [][]byte{[]byte("signer0"), []byte("signer1")}

	t.Run("transient collection -> success", func(t *testing.T) {
		policyEnvelope := policydsl.Envelope(policydsl.Or(policydsl.SignedBy(0), policydsl.SignedBy(1)), signers)
		err := ValidateConfig(createTransientCollectionConfig(coll1, policyEnvelope, 1, 2, "1m"))
		assert.NoError(t, err)
	})

	t.Run("transient collection req == 0 -> error", func(t *testing.T) {
		policyEnvelope := policydsl.Envelope(policydsl.Or(policydsl.SignedBy(0), policydsl.SignedBy(1)), signers)
		err := ValidateConfig(createTransientCollectionConfig(coll1, policyEnvelope, 0, 2, "1m"))
		require.Error(t, err)
		expectedErr := "required peer count must be greater than 0"
		assert.Truef(t, strings.Contains(err.Error(), expectedErr), "Expected error to contain '%s' but got '%s'", expectedErr, err)
	})

	t.Run("transient collection req > max -> error", func(t *testing.T) {
		policyEnvelope := policydsl.Envelope(policydsl.Or(policydsl.SignedBy(0), policydsl.SignedBy(1)), signers)
		err := ValidateConfig(createTransientCollectionConfig(coll1, policyEnvelope, 3, 2, "1m"))
		require.Error(t, err)
		expectedErr := "maximum peer count (2) must be greater than or equal to required peer count (3)"
		assert.Truef(t, strings.Contains(err.Error(), expectedErr), "Expected error to contain '%s' but got '%s'", expectedErr, err)
	})

	t.Run("transient collection no time-to-live -> error", func(t *testing.T) {
		policyEnvelope := policydsl.Envelope(policydsl.Or(policydsl.SignedBy(0), policydsl.SignedBy(1)), signers)
		err := ValidateConfig(createTransientCollectionConfig(coll1, policyEnvelope, 1, 2, ""))
		require.Error(t, err)
		expectedErr := "time to live must be specified"
		assert.Truef(t, strings.Contains(err.Error(), expectedErr), "Expected error to contain '%s' but got '%s'", expectedErr, err)
	})

	t.Run("transient collection invalid time-to-live -> error", func(t *testing.T) {
		policyEnvelope := policydsl.Envelope(policydsl.Or(policydsl.SignedBy(0), policydsl.SignedBy(1)), signers)
		err := ValidateConfig(createTransientCollectionConfig(coll1, policyEnvelope, 1, 2, "1k"))
		require.Error(t, err)
		expectedErr := "invalid time format for time to live"
		assert.Truef(t, strings.Contains(err.Error(), expectedErr), "Expected error to contain '%s' but got '%s'", expectedErr, err)
	})

	t.Run("transient collection with blocks-to-live -> error", func(t *testing.T) {
		policyEnvelope := policydsl.Envelope(policydsl.Or(policydsl.SignedBy(0), policydsl.SignedBy(1)), signers)
		config := createTransientCollectionConfig(coll1, policyEnvelope, 1, 2, "1m")
		config.BlockToLive = 100
		err := ValidateConfig(config)
		require.Error(t, err)
		expectedErr := "block-to-live not supported"
		assert.Truef(t, strings.Contains(err.Error(), expectedErr), "Expected error to contain '%s' but got '%s'", expectedErr, err)
	})
}

func createTransientCollectionConfig(collectionName string, signaturePolicyEnvelope *common.SignaturePolicyEnvelope,
	requiredPeerCount int32, maximumPeerCount int32, ttl string) *pb.StaticCollectionConfig {
	signaturePolicy := &pb.CollectionPolicyConfig_SignaturePolicy{
		SignaturePolicy: signaturePolicyEnvelope,
	}

	return &pb.StaticCollectionConfig{
		Name:              collectionName,
		Type:              pb.CollectionType_COL_TRANSIENT,
		MemberOrgsPolicy:  &pb.CollectionPolicyConfig{Payload: signaturePolicy},
		RequiredPeerCount: requiredPeerCount,
		MaximumPeerCount:  maximumPeerCount,
		TimeToLive:        ttl,
	}
}
