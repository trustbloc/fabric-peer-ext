/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package notifier

import (
	"github.com/hyperledger/fabric-protos-go/ledger/rwset/kvrwset"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/extensions/endorser/api"
	xgossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/hyperledger/fabric/extensions/roles"

	extqueryexecutor "github.com/trustbloc/fabric-peer-ext/pkg/common/simplequeryexecutor"
	extstatedb "github.com/trustbloc/fabric-peer-ext/pkg/statedb"
)

var logger = flogging.MustGetLogger("ext_cc")

const lifecycleNamespace = "_lifecycle"

type stateDBProvider interface {
	StateDBForChannel(channelID string) extstatedb.StateDB
}

type updateHandler interface {
	HandleStateUpdates(trigger *ledger.StateUpdateTrigger) error
}

// Notifier notifies the update handler of any chaincode updates
type Notifier struct {
	stateDBProvider
	api.BlockPublisherProvider
	updateHandler
}

// New creates a new chaincode update notifier
func New(handler updateHandler, bpp api.BlockPublisherProvider, stateDBProvider stateDBProvider) *Notifier {
	logger.Infof("Creating chaincode update notifier")

	return &Notifier{
		updateHandler:          handler,
		stateDBProvider:        stateDBProvider,
		BlockPublisherProvider: bpp,
	}
}

// ChannelJoined is called when a peer joins a channel
func (ci *Notifier) ChannelJoined(channelID string) {
	if roles.IsCommitter() {
		return
	}

	logger.Infof("[%s] Adding writes handlers for chaincode updates", channelID)

	cci := &channelNotifier{
		channelID:     channelID,
		updateHandler: ci.updateHandler,
		queryExecutor: extqueryexecutor.New(ci.stateDBProvider.StateDBForChannel(channelID)),
	}

	ci.ForChannel(channelID).AddWriteHandler(cci.handleWrite)
	ci.ForChannel(channelID).AddCollHashWriteHandler(cci.handleHashWrite)
}

type channelNotifier struct {
	queryExecutor ledger.SimpleQueryExecutor
	channelID     string
	updateHandler
}

func (cci channelNotifier) handleWrite(metadata xgossipapi.TxMetadata, namespace string, write *kvrwset.KVWrite) error {
	if namespace != lifecycleNamespace {
		return nil
	}

	logger.Debugf("[%s] Handling write in block [%d] and TxID [%s] - Key [%s]", cci.channelID, metadata.BlockNum, metadata.TxID, write.Key)

	return cci.HandleStateUpdates(&ledger.StateUpdateTrigger{
		LedgerID:                cci.channelID,
		CommittingBlockNum:      metadata.BlockNum,
		PostCommitQueryExecutor: cci.queryExecutor,
		StateUpdates: map[string]*ledger.KVStateUpdates{
			namespace: {
				PublicUpdates: []*kvrwset.KVWrite{write},
			},
		},
	})
}

func (cci channelNotifier) handleHashWrite(metadata xgossipapi.TxMetadata, namespace, collection string, kvWrite *kvrwset.KVWriteHash) error {
	if namespace != lifecycleNamespace {
		return nil
	}

	logger.Debugf("[%s] Handling collection hash write to collection [%s] in block [%d] and TxID [%s]", cci.channelID, collection, metadata.BlockNum, metadata.TxID)

	return cci.HandleStateUpdates(&ledger.StateUpdateTrigger{
		LedgerID:                cci.channelID,
		CommittingBlockNum:      metadata.BlockNum,
		PostCommitQueryExecutor: cci.queryExecutor,
		StateUpdates: map[string]*ledger.KVStateUpdates{
			namespace: {
				CollHashUpdates: map[string][]*kvrwset.KVWriteHash{collection: {kvWrite}},
			},
		},
	})
}
