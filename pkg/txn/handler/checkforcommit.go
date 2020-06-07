/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package handler

import (
	"github.com/golang/protobuf/proto"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/pkg/errors"

	"github.com/trustbloc/fabric-peer-ext/pkg/txn/api"
)

var logger = flogging.MustGetLogger("ext_txn")

//NewCheckForCommitHandler returns a handler that check if there is need to commit
func NewCheckForCommitHandler(rwSetIgnoreNameSpace []api.Namespace, commitType api.CommitType, next ...invoke.Handler) *CheckForCommitHandler {
	return newCheckForCommitHandler(rwSetIgnoreNameSpace, commitType, proto.Unmarshal, next...)
}

func newCheckForCommitHandler(rwSetIgnoreNameSpace []api.Namespace, commitType api.CommitType, protoUnmarshal func(buf []byte, pb proto.Message) error, next ...invoke.Handler) *CheckForCommitHandler {
	return &CheckForCommitHandler{
		rwSetIgnoreNameSpace: rwSetIgnoreNameSpace,
		commitType:           commitType,
		next:                 getNext(next),
		protoUnmarshal:       protoUnmarshal,
	}
}

//CheckForCommitHandler for checking need to commit
type CheckForCommitHandler struct {
	next                 invoke.Handler
	rwSetIgnoreNameSpace []api.Namespace
	commitType           api.CommitType
	ShouldCommit         bool
	protoUnmarshal       func(buf []byte, pb proto.Message) error
}

//Handle for endorsing transactions
func (h *CheckForCommitHandler) Handle(requestContext *invoke.RequestContext, clientContext *invoke.ClientContext) {
	txID := string(requestContext.Response.TransactionID)

	if h.commitType == api.NoCommit {
		logger.Debugf("[txID %s] No commit is necessary since commit type is [%s]", txID, h.commitType)
		return
	}

	if h.commitType == api.Commit {
		logger.Debugf("[txID %s] Commit is necessary since commit type is [%s]", txID, h.commitType)
		h.ShouldCommit = true
		if h.next != nil {
			h.next.Handle(requestContext, clientContext)
		}
		return
	}

	h.commitIfHasWriteSet(txID, requestContext, clientContext)
}

func (h *CheckForCommitHandler) commitIfHasWriteSet(txID string, requestContext *invoke.RequestContext, clientContext *invoke.ClientContext) {
	var err error

	ccAction, err := h.getChaincodeAction(requestContext)
	if err != nil {
		requestContext.Error = err
		return
	}

	if len(ccAction.Events) > 0 {
		logger.Debugf("[txID %s] Commit is necessary since commit type is [%s] and chaincode event exists in proposal response", txID, api.CommitOnWrite)
		h.ShouldCommit = true
	} else {
		txRWSet := &rwsetutil.TxRwSet{}
		if err = txRWSet.FromProtoBytes(ccAction.Results); err != nil {
			requestContext.Error = errors.WithMessage(err, "Error unmarshalling to txRWSet")
			return
		}
		if h.hasWriteSet(txRWSet, txID) {
			logger.Debugf("[txID %s] Commit is necessary since commit type is [%s] and write set exists in proposal response", txID, api.CommitOnWrite)
			h.ShouldCommit = true
		}
	}

	if h.ShouldCommit {
		if h.next != nil {
			h.next.Handle(requestContext, clientContext)
		}
	} else {
		logger.Debugf("[txID %s] Commit is NOT necessary since commit type is [%s] and NO write set exists in proposal response", txID, api.CommitOnWrite)
	}
}

func (h *CheckForCommitHandler) hasWriteSet(txRWSet *rwsetutil.TxRwSet, txID string) bool {
	for _, nsRWSet := range txRWSet.NsRwSets {
		if ignoreCC(h.rwSetIgnoreNameSpace, nsRWSet.NameSpace) {
			// Ignore this writeset
			logger.Debugf("[txID %s] Ignoring writes to [%s]", txID, nsRWSet.NameSpace)
			continue
		}
		if nsRWSet.KvRwSet != nil && len(nsRWSet.KvRwSet.Writes) > 0 {
			logger.Debugf("[txID %s] Found writes to CC [%s]. A commit will be required.", txID, nsRWSet.NameSpace)
			return true
		}

		for _, collRWSet := range nsRWSet.CollHashedRwSets {
			if ignoreCollection(h.rwSetIgnoreNameSpace, nsRWSet.NameSpace, collRWSet.CollectionName) {
				// Ignore this writeset
				logger.Debugf("[txID %s] Ignoring writes to private data collection [%s] in CC [%s]", txID, collRWSet.CollectionName, nsRWSet.NameSpace)
				continue
			}
			if collRWSet.HashedRwSet != nil && len(collRWSet.HashedRwSet.HashedWrites) > 0 {
				logger.Debugf("[txID %s] Found writes to private data collection [%s] in CC [%s]. A commit will be required.", txID, collRWSet.CollectionName, nsRWSet.NameSpace)
				return true
			}
		}
	}
	return false
}
func (h *CheckForCommitHandler) getChaincodeAction(requestContext *invoke.RequestContext) (*pb.ChaincodeAction, error) {
	// let's unmarshall one of the proposal responses to see if commit is needed
	prp := &pb.ProposalResponsePayload{}

	responses := requestContext.Response.Responses

	if len(responses) == 0 || responses[0] == nil || responses[0].ProposalResponse == nil || responses[0].ProposalResponse.Payload == nil {
		return nil, errors.New("No proposal response payload")
	}

	if err := h.protoUnmarshal(responses[0].ProposalResponse.Payload, prp); err != nil {
		return nil, errors.WithMessage(err, "Error unmarshalling to ProposalResponsePayload")
	}

	ccAction := &pb.ChaincodeAction{}
	if err := h.protoUnmarshal(prp.Extension, ccAction); err != nil {
		return nil, errors.WithMessage(err, "Error unmarshalling to ChaincodeAction")
	}

	return ccAction, nil
}

func ignoreCC(namespaces []api.Namespace, ccName string) bool {
	for _, ns := range namespaces {
		if ns.Name == ccName {
			// Ignore entire chaincode only if no collections specified
			return len(ns.Collections) == 0
		}
	}
	return false
}

func ignoreCollection(namespaces []api.Namespace, ccName, collName string) bool {
	for _, ns := range namespaces {
		if ns.Name == ccName && contains(ns.Collections, collName) {
			return true
		}
	}
	return false
}

func contains(namespaces []string, name string) bool {
	for _, ns := range namespaces {
		if ns == name {
			return true
		}
	}
	return false
}

func getNext(next []invoke.Handler) invoke.Handler {
	if len(next) > 0 {
		return next[0]
	}
	return nil
}
