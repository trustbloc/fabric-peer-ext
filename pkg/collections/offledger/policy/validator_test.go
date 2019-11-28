/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package policy

import (
	"strings"
	"testing"

	cb "github.com/hyperledger/fabric-protos-go/common"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateConfig(t *testing.T) {
	coll1 := "mycollection"

	var signers = [][]byte{[]byte("signer0"), []byte("signer1")}

	t.Run("Unsupported collection type -> fail", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		err := ValidateConfig(createOffLedgerCollectionConfig(pb.CollectionType_COL_PRIVATE, coll1, policyEnvelope, 1, 2, "1m"))
		assert.Error(t, err)
	})

	t.Run("Off-Ledger collection -> success", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		err := ValidateConfig(createOffLedgerCollectionConfig(pb.CollectionType_COL_OFFLEDGER, coll1, policyEnvelope, 1, 2, "1m"))
		assert.NoError(t, err)
	})

	t.Run("DCAS collection -> success", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		err := ValidateConfig(createOffLedgerCollectionConfig(pb.CollectionType_COL_DCAS, coll1, policyEnvelope, 1, 2, "1m"))
		assert.NoError(t, err)
	})

	t.Run("transient collection req > max -> error", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		err := ValidateConfig(createOffLedgerCollectionConfig(pb.CollectionType_COL_OFFLEDGER, coll1, policyEnvelope, 3, 2, "1m"))
		require.Error(t, err)
		expectedErr := "maximum peer count (2) must be greater than or equal to required peer count (3)"
		assert.Truef(t, strings.Contains(err.Error(), expectedErr), "Expected error to contain '%s' but got '%s'", expectedErr, err)
	})

	t.Run("Off-Ledger no time-to-live -> success", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		err := ValidateConfig(createOffLedgerCollectionConfig(pb.CollectionType_COL_OFFLEDGER, coll1, policyEnvelope, 1, 2, ""))
		require.NoError(t, err)
	})

	t.Run("Off-Ledger invalid time-to-live -> error", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		err := ValidateConfig(createOffLedgerCollectionConfig(pb.CollectionType_COL_OFFLEDGER, coll1, policyEnvelope, 1, 2, "1k"))
		require.Error(t, err)
		expectedErr := "invalid time format for time to live"
		assert.Truef(t, strings.Contains(err.Error(), expectedErr), "Expected error to contain '%s' but got '%s'", expectedErr, err)
	})

	t.Run("Off-Ledger with blocks-to-live -> error", func(t *testing.T) {
		policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)
		config := createOffLedgerCollectionConfig(pb.CollectionType_COL_OFFLEDGER, coll1, policyEnvelope, 1, 2, "1m")
		config.BlockToLive = 100
		err := ValidateConfig(config)
		require.Error(t, err)
		expectedErr := "block-to-live not supported"
		assert.Truef(t, strings.Contains(err.Error(), expectedErr), "Expected error to contain '%s' but got '%s'", expectedErr, err)
	})
}

func createOffLedgerCollectionConfig(collType pb.CollectionType, collectionName string, signaturePolicyEnvelope *cb.SignaturePolicyEnvelope,
	requiredPeerCount int32, maximumPeerCount int32, ttl string) *pb.StaticCollectionConfig {
	signaturePolicy := &pb.CollectionPolicyConfig_SignaturePolicy{
		SignaturePolicy: signaturePolicyEnvelope,
	}

	return &pb.StaticCollectionConfig{
		Name:              collectionName,
		Type:              collType,
		MemberOrgsPolicy:  &pb.CollectionPolicyConfig{Payload: signaturePolicy},
		RequiredPeerCount: requiredPeerCount,
		MaximumPeerCount:  maximumPeerCount,
		TimeToLive:        ttl,
	}
}
