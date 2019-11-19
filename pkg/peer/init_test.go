/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package peer

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/resource"
)

func TestInitialize(t *testing.T) {
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
