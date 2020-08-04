#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

@all
@lc_e2e
Feature:
  Background: Setup
    Given the channel "mychannel" is created and all peers have joined

  @lc_e2e_s1
  Scenario: e2e
    And collection config "privColl" is defined for collection "collection3" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and blocksToLive=3

    And we wait 3 seconds

    And peer "peer0.org1.example.com" is queried for installed chaincodes
    And the JSON path "#.PackageID" of the response does not contain "${e2ePackageID}"

    And chaincode "e2e_cc" is installed from path "fixtures/testdata/chaincode/e2e_cc" to all peers
    And the response is saved to variable "e2ePackageID"

    And peer "peer0.org1.example.com" is queried for installed chaincodes
    And the JSON path "#.Label" of the response contains "e2e_cc"
    And the JSON path "#.PackageID" of the response contains "${e2ePackageID}"

    And peer "peer1.org1.example.com" is queried for installed chaincodes
    And the JSON path "#.Label" of the response contains "e2e_cc"
    And the JSON path "#.PackageID" of the response contains "${e2ePackageID}"

    And peer "peer0.org2.example.com" is queried for installed chaincodes
    And the JSON path "#.Label" of the response contains "e2e_cc"
    And the JSON path "#.PackageID" of the response contains "${e2ePackageID}"

    And peer "peer1.org2.example.com" is queried for installed chaincodes
    And the JSON path "#.Label" of the response contains "e2e_cc"
    And the JSON path "#.PackageID" of the response contains "${e2ePackageID}"

    And chaincode "e2e_cc", version "v1", package ID "${e2ePackageID}", sequence 1 is checked for readiness by orgs "peerorg1,peerorg2" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy "privColl"
    And the JSON path "Org1MSP" of the boolean response equals "false"
    And the JSON path "Org2MSP" of the boolean response equals "false"

    And chaincode "e2e_cc", version "v1", package ID "${e2ePackageID}", sequence 1 is approved by orgs "peerorg1" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy "privColl"
    Then we wait 5 seconds

    And chaincode "e2e_cc", version "v1", package ID "${e2ePackageID}", sequence 1 is checked for readiness by orgs "peerorg1" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy "privColl"
    And the JSON path "Org1MSP" of the boolean response equals "true"
    And the JSON path "Org2MSP" of the boolean response equals "false"

    And chaincode "e2e_cc", version "v1", package ID "${e2ePackageID}", sequence 1 is approved by orgs "peerorg2" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy "privColl"
    Then we wait 5 seconds

    And chaincode "e2e_cc", version "v1", package ID "${e2ePackageID}", sequence 1 is checked for readiness by orgs "peerorg1" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy "privColl"
    And the JSON path "Org1MSP" of the boolean response equals "true"
    And the JSON path "Org2MSP" of the boolean response equals "true"

    Then chaincode "e2e_cc", version "v1", sequence 1 is committed by orgs "peerorg1,peerorg2" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy "privColl"
    Then we wait 5 seconds

    And committed chaincode "e2e_cc" is queried by orgs "peerorg1,peerorg2" on the "mychannel" channel
    And the JSON path "#" of the response has 1 items
    And the JSON path "0.Name" of the response equals "e2e_cc"
    And the JSON path "0.Version" of the response equals "v1"
    And the JSON path "0.Approvals.Org1MSP" of the boolean response equals "true"
    And the JSON path "0.Approvals.Org2MSP" of the boolean response equals "true"

    And all committed chaincodes are queried by orgs "peerorg1,peerorg2" on the "mychannel" channel
    And the JSON path "#.Name" of the response contains "e2e_cc"

    Then peer "peer0.org1.example.com" is queried for approved chaincode "e2e_cc" and sequence 1 on the "mychannel" channel
    And the JSON path "Name" of the response equals "e2e_cc"
    And the JSON path "Version" of the response equals "v1"
    And the JSON path "PackageID" of the response equals "${e2ePackageID}"

    And chaincode "e2e_cc" is warmed up on all peers on the "mychannel" channel

    # We need to wait a while so that all of the peers' channel membership is Gossip'ed to all other peers.
    Then we wait 20 seconds

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
    And client queries chaincode "e2e_cc" with args "getprivate,collection3,pvtKey" on the "mychannel" channel
    Then response from "e2e_cc" to client equal value "pvtVal"

    # Update the value to ensure that the state cache is updated/invalidated
    When client invokes chaincode "e2e_cc" with args "putprivate,collection3,pvtKey,pvtVal2" on the "mychannel" channel
    And we wait 2 seconds
    And client queries chaincode "e2e_cc" with args "getprivate,collection3,pvtKey" on the "mychannel" channel
    Then response from "e2e_cc" to client equal value "pvtVal2"

    When client invokes chaincode "e2e_cc" with args "delprivate,collection3,pvtKey" on the "mychannel" channel
    And we wait 2 seconds
    And client queries chaincode "e2e_cc" with args "getprivate,collection3,pvtKey" on the "mychannel" channel
    Then response from "e2e_cc" to client equal value ""

  # Auth filter test
    # In this test the example Auth filter intercepts the request and returns an error with a message that includes the channel peer endpoints
    When client queries chaincode "e2e_cc" with args "authFilterError" on the "mychannel" channel then the error response should contain "Peers in channel [mychannel]"
