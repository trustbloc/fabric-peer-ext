/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"github.com/hyperledger/fabric/extensions/roles"

	"github.com/trustbloc/fabric-peer-ext/pkg/config"
)

// IsSkipCheckForDupTxnID indicates whether or not endorsers should skip the check for duplicate transactions IDs. The check
// would still be performed during validation.
func IsSkipCheckForDupTxnID() bool {
	return config.IsSkipCheckForDupTxnID()
}

// IsPrePopulateStateCache indicates whether or not the state cache on the endorsing peer should be pre-populated
// with values retrieved from the committing peer.
func IsPrePopulateStateCache() bool {
	return roles.IsEndorser() && config.IsPrePopulateStateCache()
}

// IsSaveCacheUpdates indicates whether or not state updates should be saved on the committing peer.
func IsSaveCacheUpdates() bool {
	return roles.IsCommitter() && config.IsPrePopulateStateCache()
}
