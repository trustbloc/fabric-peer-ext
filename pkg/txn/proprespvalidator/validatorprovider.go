/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package proprespvalidator

import (
	"github.com/bluele/gcache"
	"github.com/hyperledger/fabric/core/chaincode/lifecycle"
	"github.com/hyperledger/fabric/core/committer/txvalidator/v14"
	xgossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/api"
)

//go:generate counterfeiter -o ../mocks/ccinfoprovider.gen.go --fake-name LifecycleCCInfoProvider . lifecycleCCInfoProvider

type blockPublisherProvider interface {
	ForChannel(channelID string) xgossipapi.BlockPublisher
}

type lifecycleCCInfoProvider interface {
	ChaincodeInfo(channelID, name string) (*lifecycle.LocalChaincodeInfo, error)
}

// Provider is a ProposalResponseValidator provider
type Provider struct {
	common.LedgerProvider
	common.IdentityDeserializerProvider
	blockPublisherProvider blockPublisherProvider
	validators             gcache.Cache
	lcCCInfoProvider       lifecycleCCInfoProvider
}

// New returns a new ProposalResponseValidator provider
func New(ledgerProvider common.LedgerProvider, idd common.IdentityDeserializerProvider, bpp blockPublisherProvider, lcCCInfoProvider lifecycleCCInfoProvider) *Provider {
	logger.Info("Creating ProposalResponseValidator Provider")

	p := &Provider{
		LedgerProvider:               ledgerProvider,
		IdentityDeserializerProvider: idd,
		blockPublisherProvider:       bpp,
		lcCCInfoProvider:             lcCCInfoProvider,
	}

	p.validators = gcache.New(0).LoaderFunc(func(channelID interface{}) (interface{}, error) {
		return p.createValidator(channelID.(string)), nil
	}).Build()

	return p
}

// ValidatorForChannel returns the validator for the given channel
func (m *Provider) ValidatorForChannel(channelID string) api.ProposalResponseValidator {
	v, err := m.validators.Get(channelID)
	if err != nil {
		// Should never happen
		panic(err)
	}

	return v.(api.ProposalResponseValidator)
}

func (m *Provider) createValidator(channelID string) api.ProposalResponseValidator {
	logger.Infof("[%] Creating a ProposalResponseValidator")

	idd := m.IdentityDeserializerProvider.GetIdentityDeserializer(channelID)

	pe := &txvalidator.PolicyEvaluator{
		IdentityDeserializer: idd,
	}

	return newValidator(channelID, m.GetLedger(channelID), pe, idd, m.blockPublisherProvider.ForChannel(channelID), m.lcCCInfoProvider)
}
