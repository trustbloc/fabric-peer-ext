#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

@lc_all
@lc_gossip_appdata
Feature: Application-specific data over Gossip

  Background: Setup
    Given the channel "mychannel" is created and all peers have joined

    Then we wait 10 seconds

    Then chaincode "hellocc", version "v1", package ID "hellocc:v1", sequence 1 is approved and committed by orgs "peerorg1,peerorg2" on the "mychannel" channel with endorsement policy "OR('Org1MSP.member','Org2MSP.member')" and collection policy ""

    # Wait a while for the peers to distribute their membership information
    Then we wait 20 seconds

  @gossip_appdata_retrieve
  Scenario: Retrieve application-specific data from other peers using Gossip
    When client invokes chaincode "hellocc" with args "sayhello,Hello everyone!,3" on peers "peer0.org1.example.com" on the "mychannel" channel
    Then the JSON path "#" of the response has 3 items
    And the JSON path "#.Name" of the response contains "peer1.org1.example.com:7051"
    And the JSON path "#.Name" of the response contains "peer0.org2.example.com:7051"
    And the JSON path "#.Name" of the response contains "peer1.org2.example.com:7051"
    And the JSON path "0.Message" of the response equals "Hello peer0.org1.example.com:7051! I received your message 'Hello everyone!'."
    And the JSON path "1.Message" of the response equals "Hello peer0.org1.example.com:7051! I received your message 'Hello everyone!'."
    And the JSON path "2.Message" of the response equals "Hello peer0.org1.example.com:7051! I received your message 'Hello everyone!'."

    Then we wait 5 seconds
    When client queries chaincode "hellocc" with args "gethellomessage" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then the JSON path "Message" of the response equals "Hello everyone!"
