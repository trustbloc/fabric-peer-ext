// Copyright SecureKey Technologies Inc. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

module github.com/trustbloc/fabric-peer-ext

require (
	github.com/bluele/gcache v0.0.0-20190301044115-79ae3b2d8680
	github.com/golang/protobuf v1.2.0
	github.com/hyperledger/fabric v1.4.1
	github.com/hyperledger/fabric/extensions v0.0.0
	github.com/pkg/errors v0.8.1
	github.com/spf13/viper v0.0.0-20150908122457-1967d93db724
	github.com/stretchr/testify v1.3.0
	github.com/willf/bitset v1.1.9
	go.uber.org/zap v1.9.1

)

replace github.com/hyperledger/fabric => github.com/trustbloc/fabric-mod v0.0.0-20190516131545-60d0b3b2375b

replace github.com/hyperledger/fabric/extensions => ./mod/peer

replace github.com/trustbloc/fabric-peer-ext => ./
