/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package handler

import (
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/status"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/pkg/errors"
)

// Commit is a handler that commits the endorsement responses to the Orderer and aoptionally waits for a block
// event that indicates the status of the transaction.
type Commit struct {
	next        invoke.Handler
	asyncCommit bool
}

// NewCommitHandler returns a new commit handler
func NewCommitHandler(asyncCommit bool, next ...invoke.Handler) *Commit {
	return &Commit{
		asyncCommit: asyncCommit,
		next:        getNext(next),
	}
}

// Handle handles the commit
func (c *Commit) Handle(requestContext *invoke.RequestContext, clientContext *invoke.ClientContext) {
	txnID := requestContext.Response.TransactionID

	var reg fab.Registration
	var statusNotifier <-chan *fab.TxStatusEvent

	if !c.asyncCommit {
		logger.Infof("Registering for tx [%s] event", txnID)

		var err error
		reg, statusNotifier, err = clientContext.EventService.RegisterTxStatusEvent(string(txnID))
		if err != nil {
			requestContext.Error = errors.Wrap(err, "error registering for TxStatus event")
			return
		}

		defer clientContext.EventService.Unregister(reg)
	} else {
		logger.Infof("Not registering for tx [%s] event since the call is async", txnID)
	}

	if _, err := createAndSendTransaction(clientContext.Transactor, requestContext.Response.Proposal, requestContext.Response.Responses); err != nil {
		requestContext.Error = errors.Wrap(err, "createAndSendTransaction failed")
		return
	}

	if !c.asyncCommit {
		select {
		case txStatus := <-statusNotifier:
			requestContext.Response.TxValidationCode = txStatus.TxValidationCode

			if txStatus.TxValidationCode != pb.TxValidationCode_VALID {
				logger.Infof("Got invalid TxCode for tx [%s]: %s", txnID, txStatus.TxValidationCode)

				requestContext.Error = status.New(status.EventServerStatus, int32(txStatus.TxValidationCode), "received invalid transaction", nil)
				return
			}

		case <-requestContext.Ctx.Done():
			logger.Infof("Timed out or cancelled waiting for block event for tx [%s]", txnID)

			requestContext.Error = status.New(status.ClientStatus, status.Timeout.ToInt32(), "Execute didn't receive block event", nil)
			return
		}
	}

	// Delegate to next step if any
	if c.next != nil {
		c.next.Handle(requestContext, clientContext)
	}
}

func createAndSendTransaction(sender fab.Sender, proposal *fab.TransactionProposal, resps []*fab.TransactionProposalResponse) (*fab.TransactionResponse, error) {
	txnRequest := fab.TransactionRequest{
		Proposal:          proposal,
		ProposalResponses: resps,
	}

	tx, err := sender.CreateTransaction(txnRequest)
	if err != nil {
		return nil, errors.WithMessage(err, "CreateTransaction failed")
	}

	transactionResponse, err := sender.SendTransaction(tx)
	if err != nil {
		return nil, errors.WithMessage(err, "SendTransaction failed")
	}

	return transactionResponse, nil
}
