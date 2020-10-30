// Copyright SecureKey Technologies Inc. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

module github.com/trustbloc/fabric-peer-ext/test/bddtests

require (
	github.com/cucumber/godog v0.8.1
	github.com/golang/protobuf v1.3.3
	github.com/hyperledger/fabric-protos-go v0.0.0
	github.com/hyperledger/fabric-sdk-go v1.0.0-beta3.0.20201006151309-9c426dcc5096
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-ipfs-ds-help v0.1.1
	github.com/multiformats/go-multihash v0.0.13
	github.com/pkg/errors v0.8.1
	github.com/spf13/viper v1.1.1
	github.com/trustbloc/fabric-peer-test-common v0.1.5-0.20201029180125-c96c1cfc0bd0
	golang.org/x/net v0.0.0-20190620200207-3b0461eec859 // indirect
)

replace github.com/hyperledger/fabric-protos-go => github.com/trustbloc/fabric-protos-go-ext v0.1.5-0.20201007143125-b463170dba33

go 1.14
