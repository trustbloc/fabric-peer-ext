/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chaincode

import (
	"github.com/hyperledger/fabric/core/scc"
	extscc "github.com/trustbloc/fabric-peer-ext/pkg/chaincode/scc"
)

// CreateSCC returns list of System ChainCodes provided by extensions
func CreateSCC(providers ...interface{}) []scc.SelfDescribingSysCC {
	return extscc.Create(providers...)
}
