/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package peer

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/resource"
)

func TestInitialize(t *testing.T) {
	defer removeDBPath(t)

	require.NotPanics(t, Initialize)

	require.NoError(t, resource.Mgr.Initialize(
		mocks.NewBlockPublisherProvider(),
		&mocks.LedgerProvider{},
		&mocks.GossipProvider{},
		&mocks.IdentityDeserializerProvider{},
		&mocks.IdentifierProvider{},
		&mocks.IdentityProvider{},
	))
}

func removeDBPath(t testing.TB) {
	removePath(t, config.GetTransientDataLevelDBPath())
}

func removePath(t testing.TB, path string) {
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf(err.Error())
	}
}
