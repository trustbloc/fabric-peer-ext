// Copyright SecureKey Technologies Inc. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

module github.com/trustbloc/fabric-peer-ext/test/bddtests/fixtures/fabric/peer/cmd

require (
	github.com/hyperledger/fabric v2.0.0-alpha+incompatible
	github.com/spf13/viper v1.3.2
	github.com/trustbloc/fabric-peer-ext v0.0.0
)

replace github.com/hyperledger/fabric => github.com/trustbloc/fabric-mod v0.1.1-0.20191122210718-4f3ec3a01186

replace github.com/hyperledger/fabric/extensions => ../../../../../../mod/peer

replace github.com/trustbloc/fabric-peer-ext => ../../../../../..

replace github.com/spf13/viper => github.com/spf13/viper v0.0.0-20150908122457-1967d93db724

go 1.13
