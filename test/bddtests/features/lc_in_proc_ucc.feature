#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

@lc_all
@lc_in_proc_ucc
Feature: Lifecycle in-process user chaincode

  Scenario: Approve and commit in-process user chaincode
    Given the channel "mychannel" is created and all peers have joined

    # All in-process should already be pre-installed
    Then peer "peer0.org1.example.com" is queried for installed chaincodes
    And the JSON path "#.label" of the response contains "inproc_test_cc"
    And the JSON path "#.packageID" of the response contains "inproc_test_cc:v1"
    And the JSON path "#.packageID" of the response contains "inproc_test_cc:v1.1"
    And the JSON path "#.packageID" of the response contains "inproc_test_cc:v2.0"

    # The chaincode has not been approved by any org so it's not ready to be committed
    Then chaincode "inproc_test_cc", version "v1", sequence 1 is checked for readiness by orgs "peerorg1,peerorg2" on the "mychannel" channel with endorsement policy "OR('Org1MSP.member','Org2MSP.member')" and collection policy ""
    And the JSON path "Org1MSP" of the boolean response equals "false"
    And the JSON path "Org2MSP" of the boolean response equals "false"

    # Org1 approves the chaincode
    Then chaincode "inproc_test_cc", version "v1", package ID "inproc_test_cc:v1", sequence 1 is approved by orgs "peerorg1" on the "mychannel" channel with endorsement policy "OR('Org1MSP.member','Org2MSP.member')" and collection policy ""
    And we wait 5 seconds

    Then peer "peer1.org1.example.com" is queried for approved chaincode "inproc_test_cc" and sequence 1 on the "mychannel" channel
    And the JSON path "name" of the response equals "inproc_test_cc"
    And the JSON path "version" of the response equals "v1"
    And the JSON path "packageID" of the response equals "inproc_test_cc:v1"

    # The chaincode has not been approved by Org2 so it's not ready to be committed
    Then chaincode "inproc_test_cc", version "v1", sequence 1 is checked for readiness by orgs "peerorg1,peerorg2" on the "mychannel" channel with endorsement policy "OR('Org1MSP.member','Org2MSP.member')" and collection policy ""
    And the JSON path "Org1MSP" of the boolean response equals "true"
    And the JSON path "Org2MSP" of the boolean response equals "false"

    # Org2 approves the chaincode
    Then chaincode "inproc_test_cc", version "v1", package ID "inproc_test_cc:v1", sequence 1 is approved by orgs "peerorg2" on the "mychannel" channel with endorsement policy "OR('Org1MSP.member','Org2MSP.member')" and collection policy ""
    And we wait 5 seconds

    Then peer "peer1.org2.example.com" is queried for approved chaincode "inproc_test_cc" and sequence 1 on the "mychannel" channel
    And the JSON path "name" of the response equals "inproc_test_cc"
    And the JSON path "version" of the response equals "v1"
    And the JSON path "packageID" of the response equals "inproc_test_cc:v1"

    # The chaincode is ready to be committed since it's approved by Org1 and Org2
    Then chaincode "inproc_test_cc", version "v1", sequence 1 is checked for readiness by orgs "peerorg1,peerorg2" on the "mychannel" channel with endorsement policy "OR('Org1MSP.member','Org2MSP.member')" and collection policy ""
    And the JSON path "Org1MSP" of the boolean response equals "true"
    And the JSON path "Org2MSP" of the boolean response equals "true"

    # Commit the chaincode
    Then chaincode "inproc_test_cc", version "v1", sequence 1 is committed by orgs "peerorg1,peerorg2" on the "mychannel" channel with endorsement policy "OR('Org1MSP.member','Org2MSP.member')" and collection policy ""
    Then we wait 10 seconds

    # The installed chaincode should have a single reference in mychannel
    And peer "peer0.org2.example.com" is queried for installed chaincodes
    And the JSON path "#.references.mychannel.0.name" of the response contains "inproc_test_cc"
    And the JSON path "#.references.mychannel.0.version" of the response contains "v1"

    And committed chaincode "inproc_test_cc" is queried by orgs "peerorg1,peerorg2" on the "mychannel" channel
    And the JSON path "#" of the response has 1 items
    And the JSON path "0.name" of the response equals "inproc_test_cc"
    And the JSON path "0.version" of the response equals "v1"
    And the JSON path "0.approvals.Org1MSP" of the boolean response equals "true"
    And the JSON path "0.approvals.Org2MSP" of the boolean response equals "true"

    # We need to wait a while so that all of the peers' channel membership is Gossip'ed to all other peers.
    Then we wait 10 seconds

    And client queries chaincode "inproc_test_cc" with args "getversion" on the "mychannel" channel
    Then response from "inproc_test_cc" to client equal value "v1"

    # Upgrade with only a change in the revision. Only the major and minor versions are used to find a chaincode implementation.
    And chaincode "inproc_test_cc", version "v1.0.1", package ID "inproc_test_cc:v1", sequence 2 is approved by orgs "peerorg1" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy ""
    And chaincode "inproc_test_cc", version "v1.0.1", package ID "inproc_test_cc:v1", sequence 2 is approved by orgs "peerorg2" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy ""
    And we wait 5 seconds
    Then chaincode "inproc_test_cc", version "v1.0.1", sequence 2 is committed by orgs "peerorg1,peerorg2" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy ""
    And we wait 10 seconds

    And peer "peer0.org1.example.com" is queried for installed chaincodes
    And the JSON path "#.references.mychannel.0.name" of the response contains "inproc_test_cc"
    And the JSON path "#.references.mychannel.0.version" of the response contains "v1.0.1"

    And committed chaincode "inproc_test_cc" is queried by orgs "peerorg1,peerorg2" on the "mychannel" channel
    And the JSON path "#" of the response has 1 items
    And the JSON path "0.name" of the response equals "inproc_test_cc"
    And the JSON path "0.version" of the response equals "v1.0.1"
    And the JSON path "0.approvals.Org1MSP" of the boolean response equals "true"
    And the JSON path "0.approvals.Org2MSP" of the boolean response equals "true"

    And client queries chaincode "inproc_test_cc" with args "getversion" on the "mychannel" channel
    Then response from "inproc_test_cc" to client equal value "v1"

    # Upgrading with a different major and/or minor version should target a different chaincode implementation
    And chaincode "inproc_test_cc", version "v1.1.0", package ID "inproc_test_cc:v1.1", sequence 3 is approved by orgs "peerorg1" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy ""
    And chaincode "inproc_test_cc", version "v1.1.0", package ID "inproc_test_cc:v1.1", sequence 3 is approved by orgs "peerorg2" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy ""
    And we wait 5 seconds
    Then chaincode "inproc_test_cc", version "v1.1.0", sequence 3 is committed by orgs "peerorg1,peerorg2" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy ""
    And we wait 10 seconds

    And peer "peer0.org1.example.com" is queried for installed chaincodes
    And the JSON path "#.references.mychannel.0.name" of the response contains "inproc_test_cc"
    And the JSON path "#.references.mychannel.0.version" of the response contains "v1.1.0"

    And committed chaincode "inproc_test_cc" is queried by orgs "peerorg1,peerorg2" on the "mychannel" channel
    And the JSON path "#" of the response has 1 items
    And the JSON path "0.name" of the response equals "inproc_test_cc"
    And the JSON path "0.version" of the response equals "v1.1.0"
    And the JSON path "0.approvals.Org1MSP" of the boolean response equals "true"
    And the JSON path "0.approvals.Org2MSP" of the boolean response equals "true"

    And client queries chaincode "inproc_test_cc" with args "getversion" on the "mychannel" channel
    Then response from "inproc_test_cc" to client equal value "v1.1"

    # Upgrade with only a change in the revision. Only the major and minor versions are used to find a chaincode implementation.
    And chaincode "inproc_test_cc", version "v1.1.1", package ID "inproc_test_cc:v1.1", sequence 4 is approved by orgs "peerorg1" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy ""
    And chaincode "inproc_test_cc", version "v1.1.1", package ID "inproc_test_cc:v1.1", sequence 4 is approved by orgs "peerorg2" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy ""
    And we wait 5 seconds
    Then chaincode "inproc_test_cc", version "v1.1.1", sequence 4 is committed by orgs "peerorg1,peerorg2" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy ""
    And we wait 10 seconds

    And peer "peer1.org1.example.com" is queried for installed chaincodes
    And the JSON path "#.references.mychannel.0.name" of the response contains "inproc_test_cc"
    And the JSON path "#.references.mychannel.0.version" of the response contains "v1.1.1"

    And committed chaincode "inproc_test_cc" is queried by orgs "peerorg1,peerorg2" on the "mychannel" channel
    And the JSON path "#" of the response has 1 items
    And the JSON path "0.name" of the response equals "inproc_test_cc"
    And the JSON path "0.version" of the response equals "v1.1.1"
    And the JSON path "0.approvals.Org1MSP" of the boolean response equals "true"
    And the JSON path "0.approvals.Org2MSP" of the boolean response equals "true"

    And client queries chaincode "inproc_test_cc" with args "getversion" on the "mychannel" channel
    Then response from "inproc_test_cc" to client equal value "v1.1"

    # Upgrading with a different major and/or minor version should target a different chaincode implementation
    Given collection config "coll1" is defined for collection "coll1" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and blocksToLive=3
    Then chaincode "inproc_test_cc", version "v2.0.0", package ID "inproc_test_cc:v2.0", sequence 5 is approved by orgs "peerorg1" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy "coll1"
    And chaincode "inproc_test_cc", version "v2.0.0", package ID "inproc_test_cc:v2.0", sequence 5 is approved by orgs "peerorg2" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy "coll1"
    And we wait 5 seconds
    Then chaincode "inproc_test_cc", version "v2.0.0", sequence 5 is committed by orgs "peerorg1,peerorg2" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy "coll1"
    And we wait 10 seconds

    And peer "peer1.org2.example.com" is queried for installed chaincodes
    And the JSON path "#.references.mychannel.0.name" of the response contains "inproc_test_cc"
    And the JSON path "#.references.mychannel.0.version" of the response contains "v2.0.0"

    And committed chaincode "inproc_test_cc" is queried by orgs "peerorg1,peerorg2" on the "mychannel" channel
    And the JSON path "#" of the response has 1 items
    And the JSON path "0.name" of the response equals "inproc_test_cc"
    And the JSON path "0.version" of the response equals "v2.0.0"
    And the JSON path "0.approvals.Org1MSP" of the boolean response equals "true"
    And the JSON path "0.approvals.Org2MSP" of the boolean response equals "true"

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
