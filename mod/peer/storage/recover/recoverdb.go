/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package recover

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/extensions/roles"
)

var logger = flogging.MustGetLogger("extension-recover")

//DBRecoveryHandler provides extension for recover db handler
func DBRecoveryHandler(handle func() error) func() error {
	if roles.IsCommitter() {
		return handle
	}
	return func() error {
		logger.Debug("Not a committer so not recovering DB")
		return nil
	}
}
