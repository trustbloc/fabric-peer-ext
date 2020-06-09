/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package handler

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
)

// NewPreEndorsedHandler returns a handler that populates the Endorsement Response to the request context
func NewPreEndorsedHandler(endorsementResponse *channel.Response, next ...invoke.Handler) *PreEndorsedHandler {
	return &PreEndorsedHandler{endorsementResponse: endorsementResponse, next: getNext(next)}
}

// PreEndorsedHandler holds the Endorsement response
type PreEndorsedHandler struct {
	next                invoke.Handler
	endorsementResponse *channel.Response
}

// Handle for endorsing transactions
func (i *PreEndorsedHandler) Handle(requestContext *invoke.RequestContext, clientContext *invoke.ClientContext) {
	requestContext.Response = getResponse(i.endorsementResponse)
	if i.next != nil {
		i.next.Handle(requestContext, clientContext)
	}
}

func getResponse(res *channel.Response) invoke.Response {
	return invoke.Response{
		ChaincodeStatus:  res.ChaincodeStatus,
		TransactionID:    res.TransactionID,
		Responses:        res.Responses,
		TxValidationCode: res.TxValidationCode,
		Proposal:         res.Proposal,
	}
}
