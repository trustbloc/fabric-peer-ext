/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbname

import (
	"testing"

	"github.com/stretchr/testify/require"
	extdbname "github.com/trustbloc/fabric-peer-ext/pkg/common/dbname"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/dbname/mocks"
)

const name1 = "channel1_cc1$$coll1"

func TestResolve(t *testing.T) {
	extdbname.ResolverInstance.Initialize(&mocks.PeerConfig{})
	require.Equal(t, name1, Resolve(name1))
}

func TestIsRelevant(t *testing.T) {
	extdbname.ResolverInstance.Initialize(&mocks.PeerConfig{})
	require.True(t, IsRelevant(name1))
}
