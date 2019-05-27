/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	"os"
	"reflect"
	"testing"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/extensions/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore(t *testing.T) {
	channel1 := "channel1"
	channel2 := "channel2"

	f := New()
	require.NotNil(t, f)

	s1, err := f.OpenStore(channel1)
	require.NotNil(t, s1)
	assert.NoError(t, err)
	assert.Equal(t, "*storeprovider.store", reflect.TypeOf(s1).String())
	assert.Equal(t, s1, f.StoreForChannel(channel1))

	s2, err := f.OpenStore(channel2)
	require.NotNil(t, s2)
	assert.NoError(t, err)
	assert.NotEqual(t, s1, s2)
	assert.Equal(t, s2, f.StoreForChannel(channel2))

	assert.NotPanics(t, func() {
		f.Close()
	})
}

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func testMain(m *testing.M) int {
	_, _, stop := testutil.SetupExtTestEnv()

	//set the logging level to DEBUG to test debug only code
	flogging.ActivateSpec("couchdb=debug")

	viper.Set("coll.offledger.cleanupExpired.Interval", "500ms")

	//run the tests
	code := m.Run()

	//stop couchdb
	stop()

	return code
}
