// Copyright SecureKey Technologies Inc. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

module github.com/trustbloc/fabric-peer-ext/test/bddtests

require (
	github.com/btcsuite/btcutil v0.0.0-20190425235716-9e5f4b9a998d
	github.com/cucumber/godog v0.8.1
	github.com/golang/protobuf v1.3.3
	github.com/hyperledger/fabric-protos-go v0.0.0
	github.com/hyperledger/fabric-sdk-go v1.0.0-beta2.0.20200724161828-7b97fdf0b97a
	github.com/pkg/errors v0.8.1
	github.com/spf13/viper v1.1.1
	github.com/trustbloc/fabric-peer-test-common v0.1.4-0.20200724202402-afb908880a72
	golang.org/x/net v0.0.0-20190620200207-3b0461eec859 // indirect
)

replace github.com/hyperledger/fabric-protos-go => github.com/trustbloc/fabric-protos-go-ext v0.1.4-0.20200626180529-18936b36feca

go 1.14
