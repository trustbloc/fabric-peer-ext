// Copyright SecureKey Technologies Inc. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

module github.com/trustbloc/fabric-peer-ext/test/bddtests

require (
	github.com/cucumber/godog v0.8.1
	github.com/golang/protobuf v1.3.3
	github.com/hyperledger/fabric-protos-go v0.0.0
	github.com/hyperledger/fabric-sdk-go v1.0.1-0.20210201220314-86344dc25e5d
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-ipfs-ds-help v0.1.1
	github.com/multiformats/go-multihash v0.0.13
	github.com/pkg/errors v0.9.1
	github.com/spf13/viper v1.1.1
	github.com/trustbloc/fabric-peer-test-common v0.1.6
)

replace github.com/hyperledger/fabric-protos-go => github.com/trustbloc/fabric-protos-go-ext v0.1.5

go 1.14
