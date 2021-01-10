/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package state

import (
	"testing"

	"github.com/hyperledger/fabric/extensions/gossip/mocks"
	"github.com/stretchr/testify/require"

	vcommon "github.com/trustbloc/fabric-peer-ext/pkg/validation/common"
)

//go:generate counterfeiter -o ../mocks/validationprovider.gen.go --fake-name ValidationProvider . validationProvider
//go:generate counterfeiter -o ../mocks/validationctxprovider.gen.go --fake-name ValidationContextProvider . validationContextProvider

const (
	channelID = "channel1"
)

func TestValidationMgr(t *testing.T) {
	vp := &mocks.ValidationProvider{}
	cp := &mocks.ValidationContextProvider{}

	mgr := InitValidationMgr(vp, cp)
	require.NotNil(t, mgr)

	mgr.sendValidateRequest(channelID, &vcommon.ValidationRequest{})
	require.Equal(t, 1, vp.SendValidationRequestCallCount())

	mgr.validatePending(channelID, 1000)
	require.Equal(t, 1, vp.ValidatePendingCallCount())

	mgr.cancelValidation(channelID, 1000)
	require.Equal(t, 1, cp.CancelBlockValidationCallCount())
}
