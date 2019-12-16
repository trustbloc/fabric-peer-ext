/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chaincode

import (
	"github.com/hyperledger/fabric/extensions/chaincode/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/chaincode/ucc"
)

// GetUCC returns the in-process user chaincode for the given ID
func GetUCC(ccID string) (api.UserCC, bool) {
	return ucc.Get(ccID)
}
