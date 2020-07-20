// Copyright SecureKey Technologies Inc. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

module github.com/trustbloc/fabric-peer-ext/mod/peer

replace github.com/hyperledger/fabric => github.com/trustbloc/fabric-mod v0.1.4-0.20200720203922-c40e75002b4b

replace github.com/hyperledger/fabric/extensions => ./

replace github.com/trustbloc/fabric-peer-ext => ../..

replace github.com/hyperledger/fabric-protos-go => github.com/trustbloc/fabric-protos-go-ext v0.1.4-0.20200626180529-18936b36feca

replace github.com/spf13/viper2015 => github.com/spf13/viper v0.0.0-20150908122457-1967d93db724

require (
	github.com/golang/protobuf v1.3.3
	github.com/hyperledger/fabric v2.0.0+incompatible
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20200128192331-2d899240a7ed
	github.com/hyperledger/fabric-protos-go v0.0.0-20200506201313-25f6564b9ac4
	github.com/hyperledger/fabric/extensions v0.0.0
	github.com/magiconair/properties v1.8.1
	github.com/pkg/errors v0.8.1
	github.com/spf13/viper2015 v1.3.2
	github.com/stretchr/testify v1.5.1
	github.com/trustbloc/fabric-peer-ext v0.0.0
)

go 1.14
