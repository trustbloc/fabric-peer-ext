#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

@all
@gossip_appdata
Feature: Application-specific data over Gossip

  Background: Setup
    Given the channel "mychannel" is created and all peers have joined

    Then we wait 10 seconds

    And "test" chaincode "hellocc" is instantiated from path "in-process" on the "mychannel" channel with args "" with endorsement policy "OR('Org1MSP.member','Org2MSP.member')" with collection policy ""

    # Wait a while for the peers to distribute their membership information
    Then we wait 30 seconds

  @gossip_appdata_retrieve
  Scenario: Retrieve application-specific data from other peers using Gossip
    When client queries chaincode "hellocc" with args "sayhello,Hello everyone!,3" on peers "peer0.org1.example.com" on the "mychannel" channel
    Then the JSON path "#" of the response has 3 items
    And the JSON path "#.Name" of the response contains "peer1.org1.example.com:7051"
    And the JSON path "#.Name" of the response contains "peer0.org2.example.com:7051"
    And the JSON path "#.Name" of the response contains "peer1.org2.example.com:7051"
    And the JSON path "0.Message" of the response equals "Hello peer0.org1.example.com:7051! I received your message 'Hello everyone!'."
    And the JSON path "1.Message" of the response equals "Hello peer0.org1.example.com:7051! I received your message 'Hello everyone!'."
    And the JSON path "2.Message" of the response equals "Hello peer0.org1.example.com:7051! I received your message 'Hello everyone!'."
