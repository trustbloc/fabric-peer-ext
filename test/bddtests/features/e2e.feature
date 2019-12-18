#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

@all
@e2e
Feature:

  @e2e_s1
  Scenario: e2e
    Given the channel "mychannel" is created and all peers have joined
    And collection config "privColl" is defined for collection "collection3" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and blocksToLive=3
    And "test" chaincode "e2e_cc" is installed from path "github.com/trustbloc/fabric-peer-ext/test/chaincode/e2e_cc" to all peers
    And "test" chaincode "e2e_cc" is instantiated from path "github.com/trustbloc/fabric-peer-ext/test/chaincode/e2e_cc" on the "mychannel" channel with args "" with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" with collection policy "privColl"
    And chaincode "e2e_cc" is warmed up on all peers on the "mychannel" channel

    # Perform a rolling restart of all peers to ensure that the user chaincodes are re-registered
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
    Then we wait 5 seconds

    # Test transactions
    When client invokes chaincode "e2e_cc" with args "del,k1" on the "mychannel" channel
    And we wait 2 seconds
    And client queries chaincode "e2e_cc" with args "get,k1" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then response from "e2e_cc" to client equal value ""

    When client invokes chaincode "e2e_cc" with args "put,k1,20" on the "mychannel" channel
    And we wait 2 seconds
    And client queries chaincode "e2e_cc" with args "get,k1" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then response from "e2e_cc" to client equal value "20"

    When client invokes chaincode "e2e_cc" with args "put,k1,20" on the "mychannel" channel
    And we wait 2 seconds
    And client queries chaincode "e2e_cc" with args "get,k1" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then response from "e2e_cc" to client equal value "20-20"

    When client invokes chaincode "e2e_cc" with args "del,k1" on the "mychannel" channel
    And we wait 2 seconds
    And client queries chaincode "e2e_cc" with args "get,k1" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then response from "e2e_cc" to client equal value ""


    # Test private data collection transactions
    When client invokes chaincode "e2e_cc" with args "putprivate,collection3,pvtKey,pvtVal" on the "mychannel" channel
    And we wait 2 seconds
    And client queries chaincode "e2e_cc" with args "getprivate,collection3,pvtKey" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then response from "e2e_cc" to client equal value "pvtVal"

    When client invokes chaincode "e2e_cc" with args "delprivate,collection3,pvtKey" on the "mychannel" channel
    And we wait 2 seconds
    And client queries chaincode "e2e_cc" with args "getprivate,collection3,pvtKey" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then response from "e2e_cc" to client equal value ""
