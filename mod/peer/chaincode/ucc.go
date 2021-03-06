/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chaincode

import (
	"github.com/hyperledger/fabric/extensions/chaincode/api"
	"github.com/hyperledger/fabric/msp"

	"github.com/trustbloc/fabric-peer-ext/pkg/chaincode/ucc"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/implicitpolicy"
)

// GetUCC returns the in-process user chaincode that matches the given name and version.
// The user chaincode is returned when the name matches and the major and minor versions
// match. For example:
//
// Registered chaincodes:
// - ucc1 = cc1:v1
// - ucc1_1 = cc1:v1.1
// - ucc2 = cc2:v1
//
// Then the following is returned:
// cc1,v1 => ucc1, true
// cc1,v1.0 => ucc1, true
// cc1,v1.0.1 => ucc1, true
// cc1,v1.1.0 => ucc1_1, true
// cc1,v1.2.0 => nil, false
// cc2,v1.0.5 => ucc2, true
func GetUCC(name, version string) (api.UserCC, bool) {
	return ucc.Get(name, version)
}

// GetUCCByPackageID returns the in-process user chaincode for the given package ID
func GetUCCByPackageID(id string) (api.UserCC, bool) {
	return ucc.GetByPackageID(id)
}

// Chaincodes returns all registered in-process chaincodes
func Chaincodes() []api.UserCC {
	return ucc.Chaincodes()
}

// WaitForReady blocks until the chaincodes are all registered
func WaitForReady() {
	ucc.WaitForReady()
}

// GetPackageID returns the package ID of the chaincode
func GetPackageID(cc api.UserCC) string {
	return cc.Name() + ":" + cc.Version()
}

// IsValidMSP return true if the given MSP is valid for chaincode/collection policy
func IsValidMSP(mspID string, msps map[string]msp.MSP) bool {
	if mspID == implicitpolicy.ImplicitOrg {
		return true
	}

	_, ok := msps[mspID]
	return ok
}
