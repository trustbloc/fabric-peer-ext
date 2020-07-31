#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

@all
@in_proc_ucc
Feature: In-process user chaincode

  Background: Setup
    Given the channel "mychannel" is created and all peers have joined
    And "test" chaincode "inproc_test_cc" is instantiated from path "in-process" on the "mychannel" channel with args "" with endorsement policy "OR('Org1MSP.member','Org2MSP.member')" with collection policy ""

    # We need to wait a while so that all of the peers' channel membership is Gossip'ed to all other peers.
    Then we wait 10 seconds

  Scenario: Upgrade of in-process user chaincode
    And client queries chaincode "inproc_test_cc" with args "getversion" on the "mychannel" channel
    Then response from "inproc_test_cc" to client equal value "v1"

    # Upgrade with only a change in the revision. Only the major and minor versions are used to find a chaincode implementation.
    Given "test" chaincode "inproc_test_cc" is upgraded with version "v1.0.1" from path "in-process" on the "mychannel" channel with args "" with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" with collection policy ""
    And we wait 10 seconds
    And client queries chaincode "inproc_test_cc" with args "getversion" on the "mychannel" channel
    Then response from "inproc_test_cc" to client equal value "v1"

    # Upgrading with a different major and/or minor version should target a different chaincode implementation
    Given "test" chaincode "inproc_test_cc" is upgraded with version "v1.1.0" from path "in-process" on the "mychannel" channel with args "" with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" with collection policy ""
    And we wait 10 seconds
    And client queries chaincode "inproc_test_cc" with args "getversion" on the "mychannel" channel
    Then response from "inproc_test_cc" to client equal value "v1.1"

    # Upgrade with only a change in the revision. Only the major and minor versions are used to find a chaincode implementation.
    Given "test" chaincode "inproc_test_cc" is upgraded with version "v1.1.1" from path "in-process" on the "mychannel" channel with args "" with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" with collection policy ""
    And we wait 10 seconds
    And client queries chaincode "inproc_test_cc" with args "getversion" on the "mychannel" channel
    Then response from "inproc_test_cc" to client equal value "v1.1"

    # Upgrading with a different major and/or minor version should target a different chaincode implementation
    Given collection config "coll1" is defined for collection "coll1" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and blocksToLive=3
    Given "test" chaincode "inproc_test_cc" is upgraded with version "v2.0.0" from path "in-process" on the "mychannel" channel with args "" with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" with collection policy "coll1"
    And we wait 10 seconds
    And client queries chaincode "inproc_test_cc" with args "getversion" on the "mychannel" channel
    Then response from "inproc_test_cc" to client equal value "v2.0"

    # Perform a rolling restart of all peers to ensure that the in-process user chaincodes are re-registered
    Given container "peer0.org1.example.com" is stopped
    And container "peer0.org1.example.com" is started
    Then we wait 5 seconds

    Then container "peer1.org1.example.com" is stopped
    And container "peer1.org1.example.com" is started
    Then we wait 5 seconds

    Then container "peer0.org2.example.com" is stopped
    And container "peer0.org2.example.com" is started
    Then we wait 5 seconds

    Then container "peer1.org2.example.com" is stopped
    And container "peer1.org2.example.com" is started

    Then we wait 10 seconds

    And client queries chaincode "inproc_test_cc" with args "getversion" on the "mychannel" channel
    Then response from "inproc_test_cc" to client equal value "v2.0"
