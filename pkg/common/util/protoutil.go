/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go/common"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
)

// ExtractPayload extracts the payload from the envelope
func ExtractPayload(envelope *cb.Envelope) (*cb.Payload, error) {
	payload := &cb.Payload{}
	err := proto.Unmarshal(envelope.Payload, payload)
	err = errors.Wrap(err, "no payload in envelope")
	return payload, err
}

// GetTransaction unmarshals the transaction
func GetTransaction(txBytes []byte) (*pb.Transaction, error) {
	tx := &pb.Transaction{}
	err := proto.Unmarshal(txBytes, tx)
	return tx, errors.Wrap(err, "error unmarshaling Transaction")

}

// GetChaincodeActionPayload unmarshals the chaincode action bytes
func GetChaincodeActionPayload(capBytes []byte) (*pb.ChaincodeActionPayload, error) {
	cap := &pb.ChaincodeActionPayload{}
	err := proto.Unmarshal(capBytes, cap)
	return cap, errors.Wrap(err, "error unmarshaling ChaincodeActionPayload")
}
