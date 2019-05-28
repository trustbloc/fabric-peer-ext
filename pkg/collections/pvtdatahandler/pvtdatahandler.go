/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatahandler

import (
	"context"

	"github.com/hyperledger/fabric/common/flogging"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
)

var logger = flogging.MustGetLogger("ext_pvtdatahandler")

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
