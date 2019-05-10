/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cdbblkstorage

import (
	"github.com/pkg/errors"

	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protoutil"
)

func extractTxIDFromEnvelope(txEnvelope *common.Envelope) (string, error) {
	payload, err := protoutil.GetPayload(txEnvelope)
	if err != nil {
		return "", nil
	}

	payloadHeader := payload.Header
	channelHeader, err := protoutil.UnmarshalChannelHeader(payloadHeader.ChannelHeader)
	if err != nil {
		return "", err
	}

	return channelHeader.TxId, nil
}

func extractTxnEnvelopeFromBlock(block *common.Block, txID string) (*common.Envelope, error) {
	blockData := block.GetData()
	for _, txEnvelopeBytes := range blockData.GetData() {
		envelope, err := protoutil.GetEnvelopeFromBlock(txEnvelopeBytes)
		if err != nil {
			return nil, err
		}

		id, err := extractTxIDFromEnvelope(envelope)
		if err != nil {
			return nil, err
		}
		if id != txID {
			continue
		}

		txEnvelope, err := protoutil.GetEnvelopeFromBlock(txEnvelopeBytes)
		if err != nil {
			return nil, err
		}

		return txEnvelope, nil
	}

	return nil, errors.Errorf("transaction not found [%s]", txID)
}

func extractEnvelopeFromBlock(block *common.Block, tranNum uint64) (*common.Envelope, error) {
	blockData := block.GetData()
	envelopes := blockData.GetData()
	envelopesLen := uint64(len(envelopes))
	if envelopesLen-1 < tranNum {
		blockNum := block.GetHeader().GetNumber()
		return nil, errors.Errorf("transaction number is invalid [%d, %d, %d]", blockNum, envelopesLen, tranNum)
	}
	return protoutil.GetEnvelopeFromBlock(envelopes[tranNum])
}
