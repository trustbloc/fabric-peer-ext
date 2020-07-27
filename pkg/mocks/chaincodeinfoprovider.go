/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"github.com/hyperledger/fabric/core/ledger"
)

// ChaincodeInfoProvider is a mock chaincode info provider
type ChaincodeInfoProvider struct {
	ccInfoMap map[string]*ledger.DeployedChaincodeInfo
	err       error
}

// NewChaincodeInfoProvider returns a mock chaincode info provider
func NewChaincodeInfoProvider() *ChaincodeInfoProvider {
	return &ChaincodeInfoProvider{
		ccInfoMap: make(map[string]*ledger.DeployedChaincodeInfo),
	}
}

// WithData adds mock data for a given chaincode
func (p *ChaincodeInfoProvider) WithData(ccName string, ccInfo *ledger.DeployedChaincodeInfo) *ChaincodeInfoProvider {
	p.ccInfoMap[ccName] = ccInfo

	return p
}

// WithError injects an error
func (p *ChaincodeInfoProvider) WithError(err error) *ChaincodeInfoProvider {
	p.err = err

	return p
}

// ChaincodeInfo returns mock chaincode info
func (p *ChaincodeInfoProvider) ChaincodeInfo(channelName, chaincodeName string, qe ledger.SimpleQueryExecutor) (*ledger.DeployedChaincodeInfo, error) {
	if p.err != nil {
		return nil, p.err
	}

	return p.ccInfoMap[chaincodeName], nil
}
