// Copyright SecureKey Technologies Inc. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

module github.com/trustbloc/fabric-peer-ext/mod/peer

replace github.com/hyperledger/fabric => github.com/trustbloc/fabric-mod v0.0.0-20190709140030-ec45e50f7e8a

replace github.com/hyperledger/fabric/extensions => ./

replace github.com/trustbloc/fabric-peer-ext => ../..

require (
	github.com/golang/protobuf v1.3.1
	github.com/hyperledger/fabric v2.0.0-alpha+incompatible
	github.com/hyperledger/fabric/extensions v0.0.0
	github.com/magiconair/properties v1.8.1
	github.com/pkg/errors v0.8.1
	github.com/spf13/viper v0.0.0-20150908122457-1967d93db724
	github.com/stretchr/testify v1.3.0
	github.com/trustbloc/fabric-peer-ext v0.0.0
)
