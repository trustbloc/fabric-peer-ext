/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvledger

import (
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/msgs"

	"github.com/trustbloc/fabric-peer-ext/pkg/idstore"
)

// PauseChannel updates the channel status to inactive in ledgerProviders.
func PauseChannel(config *ledger.Config, ledgerID string) error {
	if err := pauseOrResumeChannel(config, ledgerID, msgs.Status_INACTIVE); err != nil {
		return err
	}
	logger.Infof("The channel [%s] has been successfully paused", ledgerID)
	return nil
}

// ResumeChannel updates the channel status to active in ledgerProviders
func ResumeChannel(config *ledger.Config, ledgerID string) error {
	if err := pauseOrResumeChannel(config, ledgerID, msgs.Status_ACTIVE); err != nil {
		return err
	}
	logger.Infof("The channel [%s] has been successfully resumed", ledgerID)
	return nil
}

func pauseOrResumeChannel(config *ledger.Config, ledgerID string, status msgs.Status) error {
	store, err := idstore.OpenIDStore(config)
	if err != nil {
		return err
	}

	defer store.Close()
	return store.UpdateLedgerStatus(ledgerID, status)
}
