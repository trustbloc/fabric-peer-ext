/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatahandler

import (
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	extpvtdatahandler "github.com/trustbloc/fabric-peer-ext/pkg/collections/pvtdatahandler"
)

// New returns a new Handler
func New(channelID string, collDataProvider storeapi.Provider) *extpvtdatahandler.Handler {
	return extpvtdatahandler.New(channelID, collDataProvider)
}
