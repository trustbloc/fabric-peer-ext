/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
)

// IsSkipCheckForDupTxnID indicates whether or not endorsers should skip the check for duplicate transactions IDs. The check
// would still be performed during validation.
func IsSkipCheckForDupTxnID() bool {
	return config.IsSkipCheckForDupTxnID()
}
