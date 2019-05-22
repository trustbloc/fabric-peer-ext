// Copyright SecureKey Technologies Inc. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

module github.com/trustbloc/fabric-peer-ext/test/bddtests

replace github.com/hyperledger/fabric => github.com/trustbloc/fabric-mod v0.0.0-20190522001819-bbf763a5c881

replace github.com/hyperledger/fabric/extensions => github.com/trustbloc/fabric-mod/extensions v0.0.0-20190510235640-ff89b7580e81

replace github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric => github.com/trustbloc/fabric-sdk-go-ext/fabric v0.0.0-20190507173756-f4404fb70f73

require (
	github.com/DATA-DOG/godog v0.7.13
	github.com/hyperledger/fabric v1.4.1
	github.com/spf13/viper v1.0.2
	github.com/sykesm/zap-logfmt v0.0.2 // indirect
	github.com/trustbloc/fabric-peer-test-common v0.0.0-20190515153005-67b33fd24540
	go.uber.org/zap v1.10.0 // indirect
)
