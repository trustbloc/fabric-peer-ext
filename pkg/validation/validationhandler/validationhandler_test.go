/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validationhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	gproto "github.com/hyperledger/fabric-protos-go/gossip"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/gossip/common"
	gdiscovery "github.com/hyperledger/fabric/gossip/discovery"
	viper "github.com/spf13/viper2015"
	"github.com/stretchr/testify/require"

	extcommon "github.com/trustbloc/fabric-peer-ext/pkg/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/txflags"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/gossip/appdata"
	gmocks "github.com/trustbloc/fabric-peer-ext/pkg/gossip/state/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
	vcommon "github.com/trustbloc/fabric-peer-ext/pkg/validation/common"
	vmocks "github.com/trustbloc/fabric-peer-ext/pkg/validation/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/validation/validationresults"
)

//go:generate counterfeiter -o ../mocks/distvalidatorprovider.gen.go --fake-name DistributedValidatorProvider . distributedValidatorProvider
//go:generate counterfeiter -o ../mocks/contextProvider.gen.go --fake-name ContextProvider . contextProvider
//go:generate counterfeiter -o ../mocks/distvalidator.gen.go --fake-name DistributedValidator ../common DistributedValidator

const (
	org1MSPID = "Org1MSP"

	p1Org1Endpoint = "p1.org1.com"
	p2Org1Endpoint = "p2.org1.com"
	p3Org1Endpoint = "p3.org1.com"

	txID1 = "tx1"
	txID2 = "tx2"
	txID3 = "tx3"
)

var (
	p1Org1PKIID = common.PKIidType("pkiid_P1O1")
	p2Org1PKIID = common.PKIidType("pkiid_P2O1")
	p3Org1PKIID = common.PKIidType("pkiid_P3O1")

	p1 = newMember(org1MSPID, p1Org1Endpoint, true, roles.ValidatorRole)
	p2 = newMember(org1MSPID, p2Org1Endpoint, false, roles.CommitterRole)
	p3 = newMember(org1MSPID, p3Org1Endpoint, false, roles.ValidatorRole)
)

// Ensure roles are initialized
var _ = roles.GetRoles()

func TestProvider_SendValidationRequest(t *testing.T) {
	resetRoles := roles.SetRole(roles.ValidatorRole)
	defer resetRoles()

	resetViper := setViper(config.ConfDistributedValidationEnabled, true)
	defer resetViper()

	bb := mocks.NewBlockBuilder(channelID, 1000)
	bb.Transaction(txID1, peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction(txID2, peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction(txID3, peer.TxValidationCode_NOT_VALIDATED)
	block := bb.Build()

	flags := txflags.New(3)

	vr := &validationresults.Results{
		BlockNumber: block.Header.Number,
		TxFlags:     flags,
	}

	t.Run("Success", func(t *testing.T) {
		p, mp := newProviderWithMocks(t, vr)
		defer p.Close()

		p.SendValidationRequest(channelID, &vcommon.ValidationRequest{
			Block: block,
		})

		time.Sleep(100 * time.Millisecond)

		require.Equal(t, 1, mp.validator.SubmitValidationResultsCallCount())

		results := mp.validator.SubmitValidationResultsArgsForCall(0)
		require.NotNil(t, results)
		require.Equal(t, vr.BlockNumber, results.BlockNumber)
		require.Equal(t, vr.TxFlags, results.TxFlags)
	})

	t.Run("Bad response", func(t *testing.T) {
		p, mp := newProviderWithMocks(t, vr)
		defer p.Close()

		vh := p.getHandler(channelID)
		require.NotNil(t, vh)

		vh.dataRetriever = &mockDataRetriever{
			t:        t,
			response: []byte("invalid"),
		}

		p.SendValidationRequest(channelID, &vcommon.ValidationRequest{
			Block: block,
		})

		time.Sleep(100 * time.Millisecond)

		require.Equal(t, 0, mp.validator.SubmitValidationResultsCallCount())
	})

	t.Run("No remote peers", func(t *testing.T) {
		p, mp := newProviderWithMocks(t, vr)
		defer p.Close()

		mp.validator.GetValidatingPeersReturns(nil, nil)

		p.SendValidationRequest(channelID, &vcommon.ValidationRequest{
			Block: block,
		})

		time.Sleep(100 * time.Millisecond)

		require.Equal(t, 0, mp.validator.SubmitValidationResultsCallCount())
	})

	t.Run("ValidationContextForBlock error -> ignore", func(t *testing.T) {
		p, mp := newProviderWithMocks(t, vr)
		defer p.Close()

		errExpected := fmt.Errorf("injected context error")

		mp.ctx.ValidationContextForBlockReturns(nil, errExpected)

		p.SendValidationRequest(channelID, &vcommon.ValidationRequest{
			Block: block,
		})

		time.Sleep(100 * time.Millisecond)

		require.Equal(t, 0, mp.validator.SubmitValidationResultsCallCount())
	})

	t.Run("GetValidatingPeers error", func(t *testing.T) {
		p, mp := newProviderWithMocks(t, vr)
		defer p.Close()

		errExpected := fmt.Errorf("injected peers error")

		mp.validator.GetValidatingPeersReturns(nil, errExpected)

		p.SendValidationRequest(channelID, &vcommon.ValidationRequest{
			Block: block,
		})

		time.Sleep(100 * time.Millisecond)

		require.Equal(t, 0, mp.validator.SubmitValidationResultsCallCount())
	})

	t.Run("PeerFilter", func(t *testing.T) {
		p, _ := newProviderWithMocks(t, vr)
		defer p.Close()

		filter := p.getHandler(channelID).peerFilter(discovery.PeerGroup{p1, p3}, 1000)

		require.True(t, filter(p1))
		require.False(t, filter(p2))
		require.True(t, filter(p3))
	})
}

func TestProvider_handleValidationRequest(t *testing.T) {
	resetRoles := roles.SetRole(roles.ValidatorRole)
	defer resetRoles()

	resetViper := setViper(config.ConfDistributedValidationEnabled, true)
	defer resetViper()

	bb := mocks.NewBlockBuilder(channelID, 1000)
	bb.Transaction(txID1, peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction(txID2, peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction(txID3, peer.TxValidationCode_NOT_VALIDATED)
	block := bb.Build()

	blockBytes, err := proto.Marshal(block)
	require.NoError(t, err)

	req := &gproto.AppDataRequest{
		Request: blockBytes,
	}

	flags := txflags.New(3)

	vr := &validationresults.Results{
		BlockNumber: block.Header.Number,
		TxFlags:     flags,
	}

	t.Run("Block number equal to local height -> validate", func(t *testing.T) {
		p, _ := newProviderWithMocks(t, vr)
		defer p.Close()

		responder := &mockResponder{}

		p.handleValidateRequest(channelID, req, responder)

		require.NotEmpty(t, responder.data)

		valResults := &validationresults.Results{}
		require.NoError(t, json.Unmarshal(responder.data, valResults))
		require.Equal(t, block.Header.Number, valResults.BlockNumber)
		require.Empty(t, valResults.Err)
		require.NotEmpty(t, valResults.Identity)
		require.NotEmpty(t, valResults.Signature)
	})

	t.Run("Block number greater than local height -> add to pending", func(t *testing.T) {
		p, mp := newProviderWithMocks(t, vr)
		defer p.Close()

		mp.bp.Height = 999

		responder := &mockResponder{}

		require.Equal(t, 0, p.getHandler(channelID).requestCache.Size())

		p.handleValidateRequest(channelID, req, responder)

		require.Empty(t, responder.data)
		require.Equal(t, 1, p.getHandler(channelID).requestCache.Size())
	})

	t.Run("Block already committed -> discard request", func(t *testing.T) {
		p, mp := newProviderWithMocks(t, vr)
		defer p.Close()

		mp.bp.Height = 1001

		responder := &mockResponder{}

		require.Equal(t, 0, p.getHandler(channelID).requestCache.Size())

		p.handleValidateRequest(channelID, req, responder)

		require.Empty(t, responder.data)
		require.Equal(t, 0, p.getHandler(channelID).requestCache.Size())
	})

	t.Run("Bad request -> ignore", func(t *testing.T) {
		p, _ := newProviderWithMocks(t, vr)
		defer p.Close()

		responder := &mockResponder{}

		req := &gproto.AppDataRequest{
			Request: []byte("invalid"),
		}

		p.handleValidateRequest(channelID, req, responder)

		require.Empty(t, responder.data)
	})

	t.Run("ValidationContextForBlock error -> ignore", func(t *testing.T) {
		p, mp := newProviderWithMocks(t, vr)
		defer p.Close()

		responder := &mockResponder{}
		errExpected := fmt.Errorf("injected context error")

		mp.ctx.ValidationContextForBlockReturns(nil, errExpected)

		p.handleValidateRequest(channelID, req, responder)

		require.Empty(t, responder.data)
	})

	t.Run("ValidatePartial error", func(t *testing.T) {
		p, mp := newProviderWithMocks(t, vr)
		defer p.Close()

		errExpected := fmt.Errorf("injected validation error")
		mp.validator.ValidatePartialReturns(nil, nil, errExpected)

		responder := &mockResponder{}

		p.handleValidateRequest(channelID, req, responder)

		require.NotEmpty(t, responder.data)

		valResults := &validationresults.Results{}
		require.NoError(t, json.Unmarshal(responder.data, valResults))
		require.Equal(t, block.Header.Number, valResults.BlockNumber)
		require.Equal(t, errExpected.Error(), valResults.Err)
	})

	t.Run("SignValidationResults error", func(t *testing.T) {
		t.Run("GetDefaultSigningIdentity error", func(t *testing.T) {
			p, mp := newProviderWithMocks(t, vr)
			defer p.Close()

			mp.ip.GetDefaultSigningIdentityReturns(nil, fmt.Errorf("injected signing identity error"))

			responder := &mockResponder{}

			p.handleValidateRequest(channelID, req, responder)

			require.NotEmpty(t, responder.data)

			valResults := &validationresults.Results{}
			require.NoError(t, json.Unmarshal(responder.data, valResults))
			require.Equal(t, block.Header.Number, valResults.BlockNumber)
			require.Empty(t, valResults.Err)
			require.Empty(t, valResults.Identity)
			require.Empty(t, valResults.Signature)
		})

		t.Run("Serialize error", func(t *testing.T) {
			p, mp := newProviderWithMocks(t, vr)
			defer p.Close()

			mp.si.SerializeReturns(nil, fmt.Errorf("injected serialize error"))

			responder := &mockResponder{}

			p.handleValidateRequest(channelID, req, responder)

			require.NotEmpty(t, responder.data)

			valResults := &validationresults.Results{}
			require.NoError(t, json.Unmarshal(responder.data, valResults))
			require.Equal(t, block.Header.Number, valResults.BlockNumber)
			require.Empty(t, valResults.Err)
			require.Empty(t, valResults.Identity)
			require.Empty(t, valResults.Signature)
		})

		t.Run("Sign error", func(t *testing.T) {
			p, mp := newProviderWithMocks(t, vr)
			defer p.Close()

			mp.si.SignReturns(nil, fmt.Errorf("injected sign error"))

			responder := &mockResponder{}

			p.handleValidateRequest(channelID, req, responder)

			require.NotEmpty(t, responder.data)

			valResults := &validationresults.Results{}
			require.NoError(t, json.Unmarshal(responder.data, valResults))
			require.Equal(t, block.Header.Number, valResults.BlockNumber)
			require.Empty(t, valResults.Err)
			require.Empty(t, valResults.Identity)
			require.Empty(t, valResults.Signature)
		})
	})
}

func TestProvider_ValidatePending(t *testing.T) {
	resetRoles := roles.SetRole(roles.ValidatorRole)
	defer resetRoles()

	resetViper := setViper(config.ConfDistributedValidationEnabled, true)
	defer resetViper()

	bb := mocks.NewBlockBuilder(channelID, 1000)
	bb.Transaction(txID1, peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction(txID2, peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction(txID3, peer.TxValidationCode_NOT_VALIDATED)
	block := bb.Build()

	flags := txflags.New(3)

	vr := &validationresults.Results{
		BlockNumber: block.Header.Number,
		TxFlags:     flags,
	}

	t.Run("Success", func(t *testing.T) {
		p, _ := newProviderWithMocks(t, vr)
		defer p.Close()

		responder := &mockResponder{}

		p.getHandler(channelID).requestCache.Add(block, responder)

		p.ValidatePending(channelID, 1000)
		require.NotEmpty(t, responder.data)

		// Should ignore request for uncached block
		p.ValidatePending(channelID, 1001)
	})

	t.Run("ValidationContextForBlock error -> ignore", func(t *testing.T) {
		p, mp := newProviderWithMocks(t, vr)
		defer p.Close()

		errExpected := fmt.Errorf("injected context error")

		mp.ctx.ValidationContextForBlockReturns(nil, errExpected)

		responder := &mockResponder{}

		p.getHandler(channelID).requestCache.Add(block, responder)

		p.ValidatePending(channelID, 1000)
		require.Empty(t, responder.data)
	})
}

func setViper(key string, value interface{}) (reset func()) {
	oldVal := viper.Get(key)
	viper.Set(key, value)

	return func() { viper.Set(key, oldVal) }
}

func newMember(mspID, endpoint string, local bool, roles ...string) *discovery.Member {
	m := &discovery.Member{
		NetworkMember: gdiscovery.NetworkMember{
			Endpoint: endpoint,
		},
		MSPID: mspID,
		Local: local,
	}
	if roles != nil {
		m.Properties = &gproto.Properties{
			Roles: roles,
		}
	}
	return m
}

type mockDataRetriever struct {
	t        *testing.T
	response []byte
}

func (m *mockDataRetriever) Retrieve(ctxt context.Context, request *appdata.Request, responseHandler appdata.ResponseHandler, allSet appdata.AllSet, opts ...appdata.Option) (extcommon.Values, error) {
	values, err := responseHandler(m.response)

	m.t.Logf("All set: %t", allSet(values))

	return values, err
}

type mockProviders struct {
	ctx       *vmocks.ContextProvider
	validator *vmocks.DistributedValidator
	bp        *mocks.MockBlockPublisher
	ip        *mocks.IdentityProvider
	si        *mocks.SigningIdentity
}

func newProviderWithMocks(t *testing.T, vr *validationresults.Results) (*Provider, *mockProviders) {
	mp := &mockProviders{}

	mp.validator = &vmocks.DistributedValidator{}
	mp.validator.GetValidatingPeersReturns(discovery.PeerGroup{p1, p2, p3}, nil)

	validatorProvider := &vmocks.DistributedValidatorProvider{}
	validatorProvider.GetValidatorForChannelReturns(mp.validator)

	gossip := mocks.NewMockGossipAdapter().
		Self(org1MSPID, mocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
		Member(org1MSPID, mocks.NewMember(p2Org1Endpoint, p2Org1PKIID, roles.CommitterRole)).
		Member(org1MSPID, mocks.NewMember(p3Org1Endpoint, p3Org1PKIID, roles.ValidatorRole))

	gp := &mocks.GossipProvider{}
	gp.GetGossipServiceReturns(gossip)

	mp.ctx = &vmocks.ContextProvider{}
	mp.ctx.ValidationContextForBlockReturns(context.Background(), nil)

	mp.bp = mocks.NewBlockPublisher()
	mp.bp.Height = 1000

	mp.si = &mocks.SigningIdentity{}
	mp.si.SerializeReturns([]byte("serialized identity"), nil)
	mp.si.SignReturns([]byte("signature"), nil)

	mp.ip = &mocks.IdentityProvider{}
	mp.ip.GetDefaultSigningIdentityReturns(mp.si, nil)

	providers := &Providers{
		AppDataHandlerRegistry:        &gmocks.AppDataHandlerRegistry{},
		DistributedValidationProvider: validatorProvider,
		GossipProvider:                gp,
		IdentityProvider:              mp.ip,
		BlockPublisherProvider:        mocks.NewBlockPublisherProvider().WithBlockPublisher(mp.bp),
		ContextProvider:               mp.ctx,
	}

	p := NewProvider(providers)
	require.NotNil(t, p)

	vh := p.getHandler(channelID)
	require.NotNil(t, vh)

	appDataResponse, err := json.Marshal(vr)
	require.NoError(t, err)

	vh.dataRetriever = &mockDataRetriever{
		t:        t,
		response: appDataResponse,
	}

	return p, mp
}
