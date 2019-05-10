package testutil

import (
	"github.com/trustbloc/fabric-peer-ext/pkg/testutil"
)

//SetupExtTestEnv creates new couchdb instance for test
//returns couchdbd address, cleanup and stop function handle.
func SetupExtTestEnv() (addr string, cleanup func(string), stop func()) {
	return testutil.SetupExtTestEnv()
}
