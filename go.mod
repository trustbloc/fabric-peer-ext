// Copyright SecureKey Technologies Inc. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

module github.com/trustbloc/fabric-peer-ext

require (
	github.com/bluele/gcache v0.0.0-20190301044115-79ae3b2d8680
	github.com/btcsuite/btcutil v0.0.0-20170419141449-a5ecb5d9547a
	github.com/golang/protobuf v1.3.2
	github.com/hyperledger/fabric v2.0.0+incompatible
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20200128192331-2d899240a7ed
	github.com/hyperledger/fabric-protos-go v0.0.0-20200124220212-e9cfc186ba7b
	github.com/hyperledger/fabric-sdk-go v1.0.0-beta1.0.20200222173625-ff3bdd738791
	github.com/hyperledger/fabric/extensions v0.0.0
	github.com/pkg/errors v0.8.1
	github.com/spf13/viper2015 v1.3.2
	github.com/stretchr/testify v1.4.0
	github.com/willf/bitset v1.1.10
	go.uber.org/zap v1.10.0
)

replace github.com/hyperledger/fabric => github.com/trustbloc/fabric-mod v0.1.3-0.20200331030958-8bc4d7fd6230

replace github.com/hyperledger/fabric/extensions => ./mod/peer

replace github.com/trustbloc/fabric-peer-ext => ./

replace github.com/hyperledger/fabric-protos-go => github.com/trustbloc/fabric-protos-go-ext v0.1.2

replace github.com/spf13/viper2015 => github.com/spf13/viper v0.0.0-20150908122457-1967d93db724

go 1.13
