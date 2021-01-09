/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validation

import (
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/core/committer/txvalidator"
	"github.com/hyperledger/fabric/core/committer/txvalidator/plugin"
	"github.com/hyperledger/fabric/core/committer/txvalidator/v20/plugindispatcher"

	"github.com/trustbloc/fabric-peer-ext/pkg/validation"
)

// NewTxValidator returns a new transaction validator
func NewTxValidator(
	channelID string,
	sem validation.Semaphore,
	cr validation.ChannelResources,
	ler validation.LedgerResources,
	lcr plugindispatcher.LifecycleResources,
	cor plugindispatcher.CollectionResources,
	pm plugin.Mapper,
	channelPolicyManagerGetter policies.ChannelPolicyManagerGetter,
	cryptoProvider bccsp.BCCSP,
) txvalidator.Validator {
	return validation.NewValidator(channelID, sem, cr, ler, lcr, cor, pm, channelPolicyManagerGetter, cryptoProvider)
}
