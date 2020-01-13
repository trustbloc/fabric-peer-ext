/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package peer

import (
	"os"
	"testing"

	"github.com/hyperledger/fabric/extensions/collections/storeprovider"
	"github.com/hyperledger/fabric/extensions/gossip/blockpublisher"
	"github.com/stretchr/testify/require"
	extconfig "github.com/trustbloc/fabric-peer-ext/pkg/config"
	statemocks "github.com/trustbloc/fabric-peer-ext/pkg/gossip/state/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/resource"
)

func TestInitialize(t *testing.T) {
	defer removeDBPath(t)

	// Ensure that the provider instances are instantiated and registered as a resource
	require.NotNil(t, blockpublisher.ProviderInstance)
	require.NotNil(t, storeprovider.NewProviderFactory())

	require.NotPanics(t, Initialize)

	require.NoError(t, resource.Mgr.Initialize(
		&mocks.LedgerProvider{},
		&mocks.GossipProvider{},
		&mocks.IdentityDeserializerProvider{},
		&mocks.IdentifierProvider{},
		&mocks.IdentityProvider{},
		&statemocks.CCEventMgrProvider{},
	))
}

func removeDBPath(t testing.TB) {
	removePath(t, extconfig.GetTransientDataLevelDBPath())
}

func removePath(t testing.TB, path string) {
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf(err.Error())
	}
}
