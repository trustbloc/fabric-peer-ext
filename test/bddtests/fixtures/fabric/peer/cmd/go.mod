// Copyright SecureKey Technologies Inc. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

module github.com/trustbloc/fabric-peer-ext/test/bddtests/fixtures/fabric/peer/cmd

require (
	github.com/Microsoft/hcsshim v0.8.9 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/hyperledger/fabric v2.0.0+incompatible
	github.com/hyperledger/fabric/extensions v0.0.0
	github.com/prometheus/procfs v0.0.5 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper2015 v1.3.2
	github.com/trustbloc/fabric-peer-ext v0.0.0
)

replace github.com/hyperledger/fabric => github.com/trustbloc/fabric-mod v0.1.4-0.20200722122421-4ab98f2911f9

replace github.com/hyperledger/fabric/extensions => ../../../../../../mod/peer

replace github.com/trustbloc/fabric-peer-ext => ../../../../../..

replace github.com/spf13/viper2015 => github.com/spf13/viper v0.0.0-20150908122457-1967d93db724

replace github.com/hyperledger/fabric-protos-go => github.com/trustbloc/fabric-protos-go-ext v0.1.4-0.20200626180529-18936b36feca

go 1.14
