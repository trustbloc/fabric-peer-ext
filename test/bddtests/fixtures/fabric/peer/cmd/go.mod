// Copyright SecureKey Technologies Inc. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

module github.com/trustbloc/fabric-peer-ext/test/bddtests/fixtures/fabric/peer/cmd

require (
	github.com/hyperledger/fabric v2.0.0-alpha+incompatible
	github.com/hyperledger/fabric/extensions v0.0.0
	github.com/spf13/viper v1.3.2
	github.com/trustbloc/fabric-peer-ext v0.0.0
)

replace github.com/hyperledger/fabric => github.com/trustbloc/fabric-mod v0.1.1-0.20191219160942-256bdd7351ad

replace github.com/hyperledger/fabric/extensions => ../../../../../../mod/peer

replace github.com/trustbloc/fabric-peer-ext => ../../../../../..

replace github.com/spf13/viper => github.com/spf13/viper v0.0.0-20150908122457-1967d93db724

replace github.com/hyperledger/fabric-protos-go => github.com/trustbloc/fabric-protos-go-ext v0.1.1-0.20191126151100-5a61374c2e1b

go 1.13
