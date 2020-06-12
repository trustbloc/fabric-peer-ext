/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package handler

import (
	"errors"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset/kvrwset"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/txn/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/handler/mocks"
)

//go:generate counterfeiter -o ./mocks/invokehandler.gen.go -fake-name InvokeHandler github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke.Handler

func TestNewCheckForCommitHandler(t *testing.T) {
	t.Run("No request", func(t *testing.T) {
		h := NewCheckForCommitHandler(nil, api.CommitOnWrite)
		require.NotNil(t, h)

		req := &invoke.RequestContext{
			Request:  invoke.Request{},
			Opts:     invoke.Opts{},
			Response: invoke.Response{},
		}

		ctx := &invoke.ClientContext{}

		h.Handle(req, ctx)
	})

	t.Run("No commit", func(t *testing.T) {
		h := NewCheckForCommitHandler(nil, api.NoCommit)
		require.NotNil(t, h)

		txRWSet := &rwsetutil.TxRwSet{
			NsRwSets: []*rwsetutil.NsRwSet{
				{
					NameSpace: "testcc",
					KvRwSet: &kvrwset.KVRWSet{
						Writes: []*kvrwset.KVWrite{
							{
								Key:      "key1",
								IsDelete: false,
								Value:    []byte("value1"),
							},
						},
					},
					CollHashedRwSets: nil,
				},
			},
		}
		rwSetBytes, err := txRWSet.ToProtoBytes()
		require.NoError(t, err)

		ccAction := &pb.ChaincodeAction{
			Results: rwSetBytes,
			Events:  nil,
			Response: &pb.Response{
				Status:  200,
				Message: "xxx",
				Payload: nil,
			},
			ChaincodeId: &pb.ChaincodeID{
				Path:    "path",
				Name:    "cc1",
				Version: "v1",
			},
		}

		ccActionBytes, err := proto.Marshal(ccAction)
		require.NoError(t, err)

		prp := &pb.ProposalResponsePayload{
			ProposalHash: []byte("hash"),
			Extension:    ccActionBytes,
		}

		prpBytes, err := proto.Marshal(prp)
		require.NoError(t, err)

		req := &invoke.RequestContext{
			Request: invoke.Request{},
			Opts:    invoke.Opts{},
			Response: invoke.Response{
				Responses: []*fab.TransactionProposalResponse{
					{
						Endorser:        "peer1",
						Status:          200,
						ChaincodeStatus: 200,
						ProposalResponse: &pb.ProposalResponse{
							Payload: prpBytes,
						},
					},
				},
			},
		}

		ctx := &invoke.ClientContext{}

		h.Handle(req, ctx)
		require.NoError(t, req.Error)
		require.False(t, h.ShouldCommit)
	})

	t.Run("Commit", func(t *testing.T) {
		h := NewCheckForCommitHandler(nil, api.Commit, &mocks.InvokeHandler{})
		require.NotNil(t, h)

		txRWSet := &rwsetutil.TxRwSet{
			NsRwSets: []*rwsetutil.NsRwSet{
				{
					NameSpace: "testcc",
					KvRwSet: &kvrwset.KVRWSet{
						Reads: []*kvrwset.KVRead{
							{
								Key: "key1",
							},
						},
					},
					CollHashedRwSets: nil,
				},
			},
		}
		rwSetBytes, err := txRWSet.ToProtoBytes()
		require.NoError(t, err)

		ccAction := &pb.ChaincodeAction{
			Results: rwSetBytes,
			Events:  nil,
			Response: &pb.Response{
				Status:  200,
				Message: "xxx",
				Payload: nil,
			},
			ChaincodeId: &pb.ChaincodeID{
				Path:    "path",
				Name:    "cc1",
				Version: "v1",
			},
		}

		ccActionBytes, err := proto.Marshal(ccAction)
		require.NoError(t, err)

		prp := &pb.ProposalResponsePayload{
			ProposalHash: []byte("hash"),
			Extension:    ccActionBytes,
		}

		prpBytes, err := proto.Marshal(prp)
		require.NoError(t, err)

		req := &invoke.RequestContext{
			Request: invoke.Request{},
			Opts:    invoke.Opts{},
			Response: invoke.Response{
				Responses: []*fab.TransactionProposalResponse{
					{
						Endorser:        "peer1",
						Status:          200,
						ChaincodeStatus: 200,
						ProposalResponse: &pb.ProposalResponse{
							Payload: prpBytes,
						},
					},
				},
			},
		}

		ctx := &invoke.ClientContext{}

		h.Handle(req, ctx)
		require.NoError(t, req.Error)
		require.True(t, h.ShouldCommit)
	})

	t.Run("Commit on write - with CC write", func(t *testing.T) {
		h := NewCheckForCommitHandler(nil, api.CommitOnWrite)
		require.NotNil(t, h)

		txRWSet := &rwsetutil.TxRwSet{
			NsRwSets: []*rwsetutil.NsRwSet{
				{
					NameSpace: "testcc",
					KvRwSet: &kvrwset.KVRWSet{
						Writes: []*kvrwset.KVWrite{
							{
								Key:      "key1",
								IsDelete: false,
								Value:    []byte("value1"),
							},
						},
					},
					CollHashedRwSets: nil,
				},
			},
		}
		rwSetBytes, err := txRWSet.ToProtoBytes()
		require.NoError(t, err)

		ccAction := &pb.ChaincodeAction{
			Results: rwSetBytes,
			Events:  nil,
			Response: &pb.Response{
				Status:  200,
				Message: "xxx",
				Payload: nil,
			},
			ChaincodeId: &pb.ChaincodeID{
				Path:    "path",
				Name:    "cc1",
				Version: "v1",
			},
		}

		ccActionBytes, err := proto.Marshal(ccAction)
		require.NoError(t, err)

		prp := &pb.ProposalResponsePayload{
			ProposalHash: []byte("hash"),
			Extension:    ccActionBytes,
		}

		prpBytes, err := proto.Marshal(prp)
		require.NoError(t, err)

		req := &invoke.RequestContext{
			Request: invoke.Request{},
			Opts:    invoke.Opts{},
			Response: invoke.Response{
				Responses: []*fab.TransactionProposalResponse{
					{
						Endorser:        "peer1",
						Status:          200,
						ChaincodeStatus: 200,
						ProposalResponse: &pb.ProposalResponse{
							Payload: prpBytes,
						},
					},
				},
			},
		}

		ctx := &invoke.ClientContext{}

		h.Handle(req, ctx)
		require.NoError(t, req.Error)
		require.True(t, h.ShouldCommit)
	})

	t.Run("Commit on write - no write", func(t *testing.T) {
		h := NewCheckForCommitHandler(nil, api.CommitOnWrite)
		require.NotNil(t, h)

		txRWSet := &rwsetutil.TxRwSet{
			NsRwSets: []*rwsetutil.NsRwSet{
				{
					NameSpace:        "testcc",
					KvRwSet:          &kvrwset.KVRWSet{},
					CollHashedRwSets: nil,
				},
			},
		}
		rwSetBytes, err := txRWSet.ToProtoBytes()
		require.NoError(t, err)

		ccAction := &pb.ChaincodeAction{
			Results: rwSetBytes,
			Events:  nil,
			Response: &pb.Response{
				Status:  200,
				Message: "xxx",
				Payload: nil,
			},
			ChaincodeId: &pb.ChaincodeID{
				Path:    "path",
				Name:    "cc1",
				Version: "v1",
			},
		}

		ccActionBytes, err := proto.Marshal(ccAction)
		require.NoError(t, err)

		prp := &pb.ProposalResponsePayload{
			ProposalHash: []byte("hash"),
			Extension:    ccActionBytes,
		}

		prpBytes, err := proto.Marshal(prp)
		require.NoError(t, err)

		req := &invoke.RequestContext{
			Request: invoke.Request{},
			Opts:    invoke.Opts{},
			Response: invoke.Response{
				Responses: []*fab.TransactionProposalResponse{
					{
						Endorser:        "peer1",
						Status:          200,
						ChaincodeStatus: 200,
						ProposalResponse: &pb.ProposalResponse{
							Payload: prpBytes,
						},
					},
				},
			},
		}

		ctx := &invoke.ClientContext{}

		h.Handle(req, ctx)
		require.NoError(t, req.Error)
		require.False(t, h.ShouldCommit)
	})

	t.Run("Commit on write - with collection write", func(t *testing.T) {
		h := NewCheckForCommitHandler(nil, api.CommitOnWrite)
		require.NotNil(t, h)

		txRWSet := &rwsetutil.TxRwSet{
			NsRwSets: []*rwsetutil.NsRwSet{
				{
					NameSpace: "testcc",
					KvRwSet: &kvrwset.KVRWSet{
						Writes: []*kvrwset.KVWrite{},
					},
					CollHashedRwSets: []*rwsetutil.CollHashedRwSet{
						{
							CollectionName: "coll1",
							HashedRwSet: &kvrwset.HashedRWSet{
								HashedWrites: []*kvrwset.KVWriteHash{
									{
										KeyHash:   []byte("hash"),
										ValueHash: []byte("value"),
									},
								},
							},
							PvtRwSetHash: []byte("hash"),
						},
					},
				},
			},
		}
		rwSetBytes, err := txRWSet.ToProtoBytes()
		require.NoError(t, err)

		ccAction := &pb.ChaincodeAction{
			Results: rwSetBytes,
			Events:  nil,
			Response: &pb.Response{
				Status:  200,
				Message: "xxx",
				Payload: nil,
			},
			ChaincodeId: &pb.ChaincodeID{
				Path:    "path",
				Name:    "cc1",
				Version: "v1",
			},
		}

		ccActionBytes, err := proto.Marshal(ccAction)
		require.NoError(t, err)

		prp := &pb.ProposalResponsePayload{
			ProposalHash: []byte("hash"),
			Extension:    ccActionBytes,
		}

		prpBytes, err := proto.Marshal(prp)
		require.NoError(t, err)

		req := &invoke.RequestContext{
			Request: invoke.Request{},
			Opts:    invoke.Opts{},
			Response: invoke.Response{
				Responses: []*fab.TransactionProposalResponse{
					{
						Endorser:        "peer1",
						Status:          200,
						ChaincodeStatus: 200,
						ProposalResponse: &pb.ProposalResponse{
							Payload: prpBytes,
						},
					},
				},
			},
		}

		ctx := &invoke.ClientContext{}

		h.Handle(req, ctx)
		require.NoError(t, req.Error)
		require.True(t, h.ShouldCommit)
	})

	t.Run("Commit on write - Ignore namespace", func(t *testing.T) {
		ignoreNS := []api.Namespace{
			{
				Name: "testcc",
			},
		}

		h := NewCheckForCommitHandler(ignoreNS, api.CommitOnWrite)
		require.NotNil(t, h)

		txRWSet := &rwsetutil.TxRwSet{
			NsRwSets: []*rwsetutil.NsRwSet{
				{
					NameSpace: "testcc",
					KvRwSet: &kvrwset.KVRWSet{
						Writes: []*kvrwset.KVWrite{
							{
								Key:      "key1",
								IsDelete: false,
								Value:    []byte("value1"),
							},
						},
					},
					CollHashedRwSets: nil,
				},
			},
		}
		rwSetBytes, err := txRWSet.ToProtoBytes()
		require.NoError(t, err)

		ccAction := &pb.ChaincodeAction{
			Results: rwSetBytes,
			Events:  nil,
			Response: &pb.Response{
				Status:  200,
				Message: "xxx",
				Payload: nil,
			},
			ChaincodeId: &pb.ChaincodeID{
				Path:    "path",
				Name:    "cc1",
				Version: "v1",
			},
		}

		ccActionBytes, err := proto.Marshal(ccAction)
		require.NoError(t, err)

		prp := &pb.ProposalResponsePayload{
			ProposalHash: []byte("hash"),
			Extension:    ccActionBytes,
		}

		prpBytes, err := proto.Marshal(prp)
		require.NoError(t, err)

		req := &invoke.RequestContext{
			Request: invoke.Request{},
			Opts:    invoke.Opts{},
			Response: invoke.Response{
				Responses: []*fab.TransactionProposalResponse{
					{
						Endorser:        "peer1",
						Status:          200,
						ChaincodeStatus: 200,
						ProposalResponse: &pb.ProposalResponse{
							Payload: prpBytes,
						},
					},
				},
			},
		}

		ctx := &invoke.ClientContext{}

		h.Handle(req, ctx)
		require.NoError(t, req.Error)
		require.False(t, h.ShouldCommit)
	})

	t.Run("Commit on write - Ignore collection", func(t *testing.T) {
		ignoreNS := []api.Namespace{
			{
				Name:        "testcc",
				Collections: []string{"coll1"},
			},
		}

		h := NewCheckForCommitHandler(ignoreNS, api.CommitOnWrite)
		require.NotNil(t, h)

		txRWSet := &rwsetutil.TxRwSet{
			NsRwSets: []*rwsetutil.NsRwSet{
				{
					NameSpace: "testcc",
					KvRwSet: &kvrwset.KVRWSet{
						Writes: []*kvrwset.KVWrite{},
					},
					CollHashedRwSets: []*rwsetutil.CollHashedRwSet{
						{
							CollectionName: "coll1",
							HashedRwSet: &kvrwset.HashedRWSet{
								HashedWrites: []*kvrwset.KVWriteHash{
									{
										KeyHash:   []byte("hash"),
										ValueHash: []byte("value"),
									},
								},
							},
							PvtRwSetHash: []byte("hash"),
						},
					},
				},
			},
		}
		rwSetBytes, err := txRWSet.ToProtoBytes()
		require.NoError(t, err)

		ccAction := &pb.ChaincodeAction{
			Results: rwSetBytes,
			Events:  nil,
			Response: &pb.Response{
				Status:  200,
				Message: "xxx",
				Payload: nil,
			},
			ChaincodeId: &pb.ChaincodeID{
				Path:    "path",
				Name:    "cc1",
				Version: "v1",
			},
		}

		ccActionBytes, err := proto.Marshal(ccAction)
		require.NoError(t, err)

		prp := &pb.ProposalResponsePayload{
			ProposalHash: []byte("hash"),
			Extension:    ccActionBytes,
		}

		prpBytes, err := proto.Marshal(prp)
		require.NoError(t, err)

		req := &invoke.RequestContext{
			Request: invoke.Request{},
			Opts:    invoke.Opts{},
			Response: invoke.Response{
				Responses: []*fab.TransactionProposalResponse{
					{
						Endorser:        "peer1",
						Status:          200,
						ChaincodeStatus: 200,
						ProposalResponse: &pb.ProposalResponse{
							Payload: prpBytes,
						},
					},
				},
			},
		}

		ctx := &invoke.ClientContext{}

		h.Handle(req, ctx)
		require.NoError(t, req.Error)
		require.False(t, h.ShouldCommit)
	})

	t.Run("Unmarshal error", func(t *testing.T) {
		errExpected := errors.New("injected unmarshal error")
		h := newCheckForCommitHandler(nil, api.CommitOnWrite, func(buf []byte, pb proto.Message) error {
			return errExpected
		})
		require.NotNil(t, h)

		ccAction := &pb.ChaincodeAction{}

		ccActionBytes, err := proto.Marshal(ccAction)
		require.NoError(t, err)

		prp := &pb.ProposalResponsePayload{
			ProposalHash: []byte("hash"),
			Extension:    ccActionBytes,
		}

		prpBytes, err := proto.Marshal(prp)
		require.NoError(t, err)

		req := &invoke.RequestContext{
			Request: invoke.Request{},
			Opts:    invoke.Opts{},
			Response: invoke.Response{
				Responses: []*fab.TransactionProposalResponse{
					{
						Endorser:        "peer1",
						Status:          200,
						ChaincodeStatus: 200,
						ProposalResponse: &pb.ProposalResponse{
							Payload: prpBytes,
						},
					},
				},
			},
		}

		ctx := &invoke.ClientContext{}

		h.Handle(req, ctx)
		require.Error(t, req.Error)
		require.Contains(t, req.Error.Error(), errExpected.Error())
	})
}
