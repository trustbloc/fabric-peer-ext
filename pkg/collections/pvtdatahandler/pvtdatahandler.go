/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatahandler

import (
	"context"
	"errors"

	"github.com/hyperledger/fabric/common/flogging"
	commonledger "github.com/hyperledger/fabric/common/ledger"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
)

var logger = flogging.MustGetLogger("ext_pvtdatahandler")

const (
	nsJoiner      = "$$"
	pvtDataPrefix = "p"
)

// Handler handles the retrieval of kevlar-defined collection types
type Handler struct {
	channelID        string
	collDataProvider storeapi.Provider
}

// New returns a new Handler
func New(channelID string, collDataProvider storeapi.Provider) *Handler {
	return &Handler{
		channelID:        channelID,
		collDataProvider: collDataProvider,
	}
}

// HandleGetPrivateData if the collection is one of the custom Kevlar collections then the private data is returned
func (h *Handler) HandleGetPrivateData(txID, ns string, config *common.StaticCollectionConfig, key string) ([]byte, bool, error) {
	switch config.Type {
	case common.CollectionType_COL_TRANSIENT:
		logger.Debugf("Collection [%s:%s] is of type TransientData. Returning transient data for key [%s]", ns, config.Name, key)
		value, err := h.getTransientData(txID, ns, config.Name, key)
		if err != nil {
			return nil, true, err
		}
		return value, true, nil
	case common.CollectionType_COL_DCAS:
		fallthrough
	case common.CollectionType_COL_OFFLEDGER:
		logger.Debugf("Collection [%s:%s] is an off-ledger store. Returning data for key [%s]", ns, config.Name, key)
		value, err := h.getData(txID, ns, config.Name, key)
		if err != nil {
			return nil, true, err
		}
		return value, true, nil
	default:
		return nil, false, nil
	}
}

// HandleGetPrivateDataMultipleKeys if the collection is one of the custom Kevlar collections then the private data is returned
func (h *Handler) HandleGetPrivateDataMultipleKeys(txID, ns string, config *common.StaticCollectionConfig, keys []string) ([][]byte, bool, error) {
	switch config.Type {
	case common.CollectionType_COL_TRANSIENT:
		logger.Debugf("Collection [%s:%s] is of type TransientData. Returning transient data for keys [%s]", ns, config.Name, keys)
		values, err := h.getTransientDataMultipleKeys(txID, ns, config.Name, keys)
		if err != nil {
			return nil, true, err
		}
		return values, true, nil
	case common.CollectionType_COL_DCAS:
		fallthrough
	case common.CollectionType_COL_OFFLEDGER:
		logger.Debugf("Collection [%s:%s] is of an off-ledger store. Returning data for keys [%s]", ns, config.Name, keys)
		values, err := h.getDataMultipleKeys(txID, ns, config.Name, keys)
		if err != nil {
			return nil, true, err
		}
		return values, true, nil
	default:
		return nil, false, nil
	}
}

// HandleExecuteQueryOnPrivateData executes the given query on the collection if the collection is one of the extended collections
func (h *Handler) HandleExecuteQueryOnPrivateData(txID, ns string, config *common.StaticCollectionConfig, query string) (commonledger.ResultsIterator, bool, error) {
	switch config.Type {
	case common.CollectionType_COL_TRANSIENT:
		logger.Debugf("Collection [%s:%s] is a TransientData store. Rich queries are not supported for transient data", ns, config.Name)
		return nil, true, errors.New("rich queries not supported on transient data")
	case common.CollectionType_COL_DCAS:
		fallthrough
	case common.CollectionType_COL_OFFLEDGER:
		logger.Debugf("Collection [%s:%s] is an off-ledger store. Returning results for query [%s]", ns, config.Name, query)
		values, err := h.executeQuery(txID, ns, config.Name, query)
		return values, true, err
	default:
		return nil, false, nil
	}
}

func (h *Handler) getTransientData(txID, ns, coll, key string) ([]byte, error) {
	ctxt, cancel := context.WithTimeout(context.Background(), config.GetTransientDataPullTimeout())
	defer cancel()

	v, err := h.collDataProvider.RetrieverForChannel(h.channelID).
		GetTransientData(ctxt, storeapi.NewKey(txID, ns, coll, key))
	if err != nil {
		return nil, err
	}

	if v == nil {
		return nil, nil
	}

	return v.Value, nil
}

func (h *Handler) getTransientDataMultipleKeys(txID, ns, coll string, keys []string) ([][]byte, error) {
	ctxt, cancel := context.WithTimeout(context.Background(), config.GetTransientDataPullTimeout())
	defer cancel()

	vals, err := h.collDataProvider.RetrieverForChannel(h.channelID).
		GetTransientDataMultipleKeys(ctxt, storeapi.NewMultiKey(txID, ns, coll, keys...))
	if err != nil {
		return nil, err
	}

	values := make([][]byte, len(vals))
	for i, v := range vals {
		if v == nil {
			values[i] = nil
		} else {
			values[i] = v.Value
		}
	}
	return values, nil
}

func (h *Handler) getData(txID, ns, coll, key string) ([]byte, error) {
	ctxt, cancel := context.WithTimeout(context.Background(), config.GetOLCollPullTimeout())
	defer cancel()

	v, err := h.collDataProvider.RetrieverForChannel(h.channelID).
		GetData(ctxt, storeapi.NewKey(txID, ns, coll, key))
	if err != nil {
		return nil, err
	}

	if v == nil {
		return nil, nil
	}

	return v.Value, nil
}

func (h *Handler) getDataMultipleKeys(txID, ns, coll string, keys []string) ([][]byte, error) {
	ctxt, cancel := context.WithTimeout(context.Background(), config.GetOLCollPullTimeout())
	defer cancel()

	vals, err := h.collDataProvider.RetrieverForChannel(h.channelID).
		GetDataMultipleKeys(ctxt, storeapi.NewMultiKey(txID, ns, coll, keys...))
	if err != nil {
		return nil, err
	}

	values := make([][]byte, len(vals))
	for i, v := range vals {
		if v == nil {
			values[i] = nil
		} else {
			values[i] = v.Value
		}
	}
	return values, nil
}

func (h *Handler) executeQuery(txID, ns, coll string, query string) (commonledger.ResultsIterator, error) {
	ctxt, cancel := context.WithTimeout(context.Background(), config.GetOLCollPullTimeout())
	defer cancel()

	it, err := h.collDataProvider.RetrieverForChannel(h.channelID).Query(ctxt, storeapi.NewQueryKey(txID, ns, coll, query))
	if err != nil {
		return nil, err
	}
	return newKVIterator(it), nil
}

type kvIterator struct {
	it storeapi.ResultsIterator
}

func newKVIterator(it storeapi.ResultsIterator) *kvIterator {
	return &kvIterator{
		it: it,
	}
}

func (kvit *kvIterator) Next() (commonledger.QueryResult, error) {
	result, err := kvit.it.Next()
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return &queryresult.KV{
		Namespace: asPvtDataNs(result.Namespace, result.Collection),
		Key:       result.Key.Key,
		Value:     result.Value,
	}, nil
}

func (kvit *kvIterator) Close() {
	kvit.it.Close()
}

func asPvtDataNs(ns, coll string) string {
	return ns + nsJoiner + pvtDataPrefix + coll
}
