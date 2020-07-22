/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvledger

import (
	"fmt"
	"testing"

	"github.com/hyperledger/fabric/common/ledger/dataformat"
	"github.com/hyperledger/fabric/core/ledger/mock"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/idstore"
)

func TestUpgradeWrongFormat(t *testing.T) {
	conf, cleanup := testConfig(t)
	conf.HistoryDBConfig.Enabled = false
	defer cleanup()
	provider := testutilNewProvider(conf, t, &mock.DeployedChaincodeInfoProvider{})

	// change format to a wrong value to test UpgradeFormat error
	env := idstore.NewTestStoreEnv(t, "", nil)
	require.NoError(t, idstore.SaveMetadataDoc(env.TestStore, "x.0"))
	provider.Close()

	err := UpgradeDBs(conf)
	expectedErr := &dataformat.ErrFormatMismatch{
		ExpectedFormat: dataformat.PreviousFormat,
		Format:         "x.0",
		DBInfo:         fmt.Sprintf("CouchDB for channel-IDs"),
	}

	require.EqualError(t, err, expectedErr.Error())
}
