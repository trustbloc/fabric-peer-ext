#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

@all
@ledger_config
Feature: ledger-config

  Background: Setup
    Given the channel "mychannel" is created and all peers have joined


  @ledger_config_s1
  Scenario: Save, get and delete application config
    # Save and query app1 v1 config
    Given variable "msp1App1V1Config" is assigned the JSON value '{"MspID":"MSP1","Apps":[{"AppName":"app1","Version":"v1","Config":"msp1-app1-v1-config","Format":"Other"}]}'
    When client invokes chaincode "configscc" with args "save,${msp1App1V1Config}" on peers "peer0.org1.example.com" on the "mychannel" channel
    And we wait 1 seconds
    Given variable "msp1App1Criteria" is assigned the JSON value '{"MspID":"MSP1","AppName":"app1"}'
    When client queries chaincode "configscc" with args "get,${msp1App1Criteria}" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then the JSON path "#" of the response has 1 items
    And the JSON path "0.Config" of the response equals "msp1-app1-v1-config"

    # Save and query app1 v2 config
    Given variable "msp1App1V2Config" is assigned the JSON value '{"MspID":"MSP1","Apps":[{"AppName":"app1","Version":"v2","Config":"msp1-app1-v2-config","Format":"Other"}]}'
    When client invokes chaincode "configscc" with args "save,${msp1App1V2Config}" on peers "peer0.org1.example.com" on the "mychannel" channel
    And we wait 1 seconds
    When client queries chaincode "configscc" with args "get,${msp1App1Criteria}" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then the JSON path "#" of the response has 2 items
    And the JSON path "#.Config" of the response contains "msp1-app1-v1-config"
    And the JSON path "#.Config" of the response contains "msp1-app1-v2-config"

    # Save and query app2 v1 & v2 config
    Given variable "msp1App2Config" is assigned the JSON value '{"MspID":"MSP1","Apps":[{"AppName":"app2","Version":"v1","Config":"msp1-app2-v1-config","Format":"Other"},{"AppName":"app2","Version":"v2","Config":"msp1-app2-v2-config","Format":"Other"}]}'
    When client invokes chaincode "configscc" with args "save,${msp1App2Config}" on peers "peer0.org1.example.com" on the "mychannel" channel
    And we wait 1 seconds
    Given variable "msp1App2Criteria" is assigned the JSON value '{"MspID":"MSP1","AppName":"app2"}'
    When client queries chaincode "configscc" with args "get,${msp1App2Criteria}" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then the JSON path "#" of the response has 2 items
    And the JSON path "#.Config" of the response contains "msp1-app2-v1-config"
    And the JSON path "#.Config" of the response contains "msp1-app2-v2-config"

    # Query all config for msp1
    Given variable "msp1Criteria" is assigned the JSON value '{"MspID":"MSP1"}'
    When client queries chaincode "configscc" with args "get,${msp1Criteria}" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then the JSON path "#" of the response has 4 items
    And the JSON path "#.Config" of the response contains "msp1-app1-v1-config"
    And the JSON path "#.Config" of the response contains "msp1-app1-v2-config"
    And the JSON path "#.Config" of the response contains "msp1-app2-v1-config"
    And the JSON path "#.Config" of the response contains "msp1-app2-v2-config"

    # Delete app1
    When client invokes chaincode "configscc" with args "delete,${msp1App1Criteria}" on peers "peer0.org1.example.com" on the "mychannel" channel
    And we wait 1 seconds
    When client queries chaincode "configscc" with args "get,${msp1Criteria}" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then the JSON path "#" of the response has 2 items
    And the JSON path "#.Config" of the response contains "msp1-app2-v1-config"
    And the JSON path "#.Config" of the response contains "msp1-app2-v2-config"

  @ledger_config_s2
  Scenario: Save, get, and delete component config
    # Save msp2, app1-v1, comp1-v1 config
    Given variable "msp2App1Comp1Config" is assigned the JSON value '{"MspID":"MSP2","Apps":[{"AppName":"app1","Version":"v1","Components":[{"Name":"comp1","Version":"v1","Config":"msp2-app1-comp1-v1-config","Format":"Other"}]}]}'
    When client invokes chaincode "configscc" with args "save,${msp2App1Comp1Config}" on peers "peer0.org1.example.com" on the "mychannel" channel
    And we wait 1 seconds
    Given variable "msp2App1Criteria" is assigned the JSON value '{"MspID":"MSP2","AppName":"app1"}'
    When client queries chaincode "configscc" with args "get,${msp2App1Criteria}" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then the JSON path "#" of the response has 1 items
    And the JSON path "0.ComponentName" of the response equals "comp1"
    And the JSON path "0.Config" of the response equals "msp2-app1-comp1-v1-config"

    # Save msp2, app1-v1, comp2-v1 config
    Given variable "msp2App1Comp2Config" is assigned the JSON value '{"MspID":"MSP2","Apps":[{"AppName":"app1","Version":"v1","Components":[{"Name":"comp2","Version":"v1","Config":"msp2-app1-comp2-v1-config","Format":"Other"}]}]}'
    When client invokes chaincode "configscc" with args "save,${msp2App1Comp2Config}" on peers "peer0.org1.example.com" on the "mychannel" channel
    And we wait 1 seconds
    Given variable "msp2App1Criteria" is assigned the JSON value '{"MspID":"MSP2","AppName":"app1"}'
    When client queries chaincode "configscc" with args "get,${msp2App1Criteria}" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then the JSON path "#" of the response has 2 items
    And the JSON path "#.Config" of the response contains "msp2-app1-comp1-v1-config"
    And the JSON path "#.Config" of the response contains "msp2-app1-comp2-v1-config"

  @ledger_config_s3
  Scenario: Ledger config service - peer-specific config
    # Save the config
    Given variable "testSccGeneralConfig" is assigned the JSON value '{"MspID":"general","Apps":[{"AppName":"testscc","Version":"v1","Components":[{"Name":"comp1","Version":"v1","Config":"{\"Org\":\"general\",\"Application\":\"testscc\",\"SubComponent\":\"comp1\"}","Format":"JSON"},{"Name":"comp2","Version":"v1","Config":"{\"Org\":\"general\",\"Application\":\"testscc\",\"SubComponent\":\"comp2\"}","Format":"JSON"}]}]}'
    Given variable "testSccOrg1Config" is assigned the JSON value '{"MspID":"Org1MSP","Peers":[{"PeerID":"peer0.org1.example.com","Apps":[{"AppName":"testscc","Version":"v1","Config":"p0-org1-testscc-v1-config","Format":"Other"}]},{"PeerID":"peer1.org1.example.com","Apps":[{"AppName":"testscc","Version":"v1","Config":"p1-org1-testscc-v1-config","Format":"Other"}]}]}'
    Given variable "testSccOrg2Config" is assigned the JSON value '{"MspID":"Org2MSP","Peers":[{"PeerID":"peer0.org2.example.com","Apps":[{"AppName":"testscc","Version":"v1","Config":"p0-org2-testscc-v1-config","Format":"Other"}]},{"PeerID":"peer1.org2.example.com","Apps":[{"AppName":"testscc","Version":"v1","Config":"p1-org2-testscc-v1-config","Format":"Other"}]}]}'
    Then client invokes chaincode "configscc" with args "save,${testSccGeneralConfig}" on peers "peer0.org1.example.com" on the "mychannel" channel
    And client invokes chaincode "configscc" with args "save,${testSccOrg1Config}" on peers "peer0.org1.example.com" on the "mychannel" channel
    And client invokes chaincode "configscc" with args "save,${testSccOrg2Config}" on peers "peer0.org2.example.com" on the "mychannel" channel
    # Wait for the transactions to commit and for config update events to fire.
    And we wait 3 seconds

    # Get general (global) data which is not specific to an org (tests retrieval of config data using the config service)
    Given variable "testSccGeneralComp1Criteria" is assigned the JSON value '{"MspID":"general","AppName":"testscc","AppVersion":"v1","ComponentName":"comp1","ComponentVersion":"v1"}'
    When client queries chaincode "testscc" with args "getconfig,${testSccGeneralComp1Criteria}" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then the JSON path "Org" of the response equals "general"
    Then the JSON path "Application" of the response equals "testscc"
    Then the JSON path "SubComponent" of the response equals "comp1"
    Given variable "testSccGeneralComp2Criteria" is assigned the JSON value '{"MspID":"general","AppName":"testscc","AppVersion":"v1","ComponentName":"comp2","ComponentVersion":"v1"}'
    When client queries chaincode "testscc" with args "getconfig,${testSccGeneralComp2Criteria}" on a single peer in the "peerorg2" org on the "mychannel" channel
    Then the JSON path "Org" of the response equals "general"
    Then the JSON path "Application" of the response equals "testscc"
    Then the JSON path "SubComponent" of the response equals "comp2"

    # Get peer-specific data (tests config update events)
    Given variable "testSccOrg1Peer0Criteria" is assigned the JSON value '{"MspID":"Org1MSP","PeerID":"peer0.org1.example.com","AppName":"testscc","AppVersion":"v1"}'
    When client queries chaincode "testscc" with args "getconfig,${testSccOrg1Peer0Criteria}" on peers "peer0.org1.example.com" on the "mychannel" channel
    Then response from "configscc" to client equal value "p0-org1-testscc-v1-config"
    Given variable "testSccOrg1Peer1Criteria" is assigned the JSON value '{"MspID":"Org1MSP","PeerID":"peer1.org1.example.com","AppName":"testscc","AppVersion":"v1"}'
    When client queries chaincode "testscc" with args "getconfig,${testSccOrg1Peer1Criteria}" on peers "peer1.org1.example.com" on the "mychannel" channel
    Then response from "configscc" to client equal value "p1-org1-testscc-v1-config"
    Given variable "testSccOrg2Peer0Criteria" is assigned the JSON value '{"MspID":"Org2MSP","PeerID":"peer0.org2.example.com","AppName":"testscc","AppVersion":"v1"}'
    When client queries chaincode "testscc" with args "getconfig,${testSccOrg2Peer0Criteria}" on peers "peer0.org2.example.com" on the "mychannel" channel
    Then response from "configscc" to client equal value "p0-org2-testscc-v1-config"
    Given variable "testSccOrg2Peer1Criteria" is assigned the JSON value '{"MspID":"Org2MSP","PeerID":"peer1.org2.example.com","AppName":"testscc","AppVersion":"v1"}'
    When client queries chaincode "testscc" with args "getconfig,${testSccOrg2Peer1Criteria}" on peers "peer1.org2.example.com" on the "mychannel" channel
    Then response from "configscc" to client equal value "p1-org2-testscc-v1-config"
    # The following peer-specific data should not be found on foreign peer
    Given variable "testSccOrg1Peer0Criteria" is assigned the JSON value '{"MspID":"Org1MSP","PeerID":"peer0.org1.example.com","AppName":"testscc","AppVersion":"v1"}'
    When client queries chaincode "testscc" with args "getconfig,${testSccOrg1Peer0Criteria}" on peers "peer1.org1.example.com" on the "mychannel" channel
    Then response from "configscc" to client equal value ""
