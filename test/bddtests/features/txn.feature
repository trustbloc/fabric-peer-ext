#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

@all
@txn
Feature: txn

  Background: Setup
    Given the channel "mychannel" is created and all peers have joined
    And "test" chaincode "configscc" is instantiated from path "in-process" on the "mychannel" channel with args "" with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" with collection policy ""

    And collection config "privColl" is defined for collection "collection3" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and blocksToLive=3
    And "test" chaincode "target_cc" is installed from path "github.com/trustbloc/fabric-peer-ext/test/chaincode/e2e_cc" to all peers
    And "test" chaincode "target_cc" is instantiated from path "github.com/trustbloc/fabric-peer-ext/test/chaincode/e2e_cc" on the "mychannel" channel with args "" with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" with collection policy "privColl"

    # We need to wait a while so that all of the peers' channel membership is Gossip'ed to all other peers.
    Then we wait 20 seconds

    Given variable "org1Config" is assigned config from file "./fixtures/config/fabric/org1-config.json"
    And variable "org2Config" is assigned config from file "./fixtures/config/fabric/org2-config.json"

    When "peerorg1" client invokes chaincode "configscc" with args "save,${org1Config}" on the "mychannel" channel
    And "peerorg2" client invokes chaincode "configscc" with args "save,${org2Config}" on the "mychannel" channel
    And we wait 5 seconds

  @txn_s1
  Scenario: Endorsements using TXN service
    Given variable "endorseAndCommitRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["put","key1","value1"],"commit_type":"commit-on-write","ignore_namespaces":[{"Name":"target_cc"}]}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorseandcommit,${endorseAndCommitRequest}" on peers "peer0.org2.example.com"
    Then the JSON path "Committed" of the boolean response equals "false"

    Given variable "endorseAndCommitRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["put","key1","value1"],"commit_type":"commit-on-write"}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorseandcommit,${endorseAndCommitRequest}" on peers "peer0.org2.example.com"
    Then the JSON path "Committed" of the boolean response equals "true"

    Given variable "endorseAndCommitRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["put","key2","value2"],"commit_type":"no-commit"}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorseandcommit,${endorseAndCommitRequest}" on peers "peer0.org2.example.com"
    Then the JSON path "Committed" of the boolean response equals "false"

    Given variable "endorseAndCommitRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["get","key1"],"commit_type":"commit-on-write"}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorseandcommit,${endorseAndCommitRequest}" on peers "peer0.org2.example.com"
    Then the JSON path "Committed" of the boolean response equals "false"

    Given variable "endorseAndCommitRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["get","key1"],"commit_type":"commit"}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorseandcommit,${endorseAndCommitRequest}" on peers "peer0.org2.example.com"
    Then the JSON path "Committed" of the boolean response equals "true"

    Given variable "endorseRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["put","key3","value3"]}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorse,${endorseRequest}" on peers "peer0.org2.example.com"
    Then the JSON path "Committed" of the boolean response equals "false"

    Then we wait 5 seconds

    Then client queries chaincode "target_cc" with args "get,key1" on the "mychannel" channel
    Then response from "target_cc" to client equal value "value1"

    Then client queries chaincode "target_cc" with args "get,key2" on the "mychannel" channel
    Then response from "target_cc" to client equal value ""

    Then client queries chaincode "target_cc" with args "get,key3" on the "mychannel" channel
    Then response from "target_cc" to client equal value ""

    Given variable "endorseAndCommitRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["put","key2","value2"]}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorseandcommit,${endorseAndCommitRequest}" on peers "peer0.org2.example.com"
    Then the JSON path "Committed" of the boolean response equals "true"

    Then we wait 5 seconds

    Then container "peer0.org2.example.com" is stopped
    And container "peer0.org2.example.com" is started

    Then container "peer1.org2.example.com" is stopped
    And container "peer1.org2.example.com" is started

    Then we wait 10 seconds

    Given variable "endorseAndCommitRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["put","key3","value3"]}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorseandcommit,${endorseAndCommitRequest}" on peers "peer1.org1.example.com"
    Then the JSON path "Committed" of the boolean response equals "true"
    Then we wait 5 seconds

    Given variable "endorseRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["get","key1"]}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorse,${endorseRequest}" on peers "peer1.org2.example.com"
    Then the JSON path "Payload" of the response equals "value1"

    Given variable "endorseRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["get","key2"]}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorse,${endorseRequest}" on peers "peer0.org1.example.com"
    Then the JSON path "Payload" of the response equals "value2"

    Given variable "endorseRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["get","key3"]}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorse,${endorseRequest}" on peers "peer0.org2.example.com"
    Then the JSON path "Payload" of the response equals "value3"

    # SDK config update
    Given variable "org1ConfigUpdate" is assigned config from file "./fixtures/config/fabric/org1-config-update.json"
    And variable "org2ConfigUpdate" is assigned config from file "./fixtures/config/fabric/org2-config-update.json"

    When "peerorg1" client invokes chaincode "configscc" with args "save,${org1ConfigUpdate}" on the "mychannel" channel
    And "peerorg2" client invokes chaincode "configscc" with args "save,${org2ConfigUpdate}" on the "mychannel" channel
    And we wait 5 seconds

    Given variable "endorseAndCommitRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["put","keyA","valueA"]}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorseandcommit,${endorseAndCommitRequest}" on peers "peer0.org1.example.com"
    Then the JSON path "Committed" of the boolean response equals "true"

    Given variable "endorseAndCommitRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["put","keyB","valueB"]}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorseandcommit,${endorseAndCommitRequest}" on peers "peer0.org2.example.com"
    Then the JSON path "Committed" of the boolean response equals "true"

    And we wait 5 seconds

    Given variable "endorseRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["get","keyA"]}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorse,${endorseRequest}" on peers "peer1.org1.example.com"
    Then the JSON path "Payload" of the response equals "valueA"

    Given variable "endorseRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["get","keyB"]}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorse,${endorseRequest}" on peers "peer1.org2.example.com"
    Then the JSON path "Payload" of the response equals "valueB"

    # Peer filter
    Given variable "endorseRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["get","keyA"],"peer_filter":"msp","peer_filter_args":["Org1MSP","Org2MSP"]}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorse,${endorseRequest}" on peers "peer1.org1.example.com"
    Then the JSON path "Payload" of the response equals "valueA"

    # Should fail since the chaincode policy won't be satisfied if we're filtering out Org2MSP
    Given variable "endorseRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["get","keyA"],"peer_filter":"msp","peer_filter_args":["Org1MSP"]}'
    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorse,${endorseRequest}" on peers "peer1.org1.example.com" then the error response should contain "no endorsement combination can be satisfied"

    # Transaction ID is provided by the client
    # - First send a request with an invalid TxnID since we don't know the identity of the peer in order to construct a valid one
    Given variable "txIDNonce" is assigned the base64 URL-encoded value "txIDNonce1"
    Given variable "endorseAndCommitRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["put","keyX","valueX"],"tx_id":"txID","nonce":"${txIDNonce}"}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorseandcommit,${endorseAndCommitRequest}" on peers "peer1.org1.example.com"
    Then the JSON path "Status" of the numeric response equals "400"
    Then the JSON path "Committed" of the boolean response equals "false"
    And the JSON path "Payload" of the response is saved to variable "identity"

    # - Now use the identity specified in the response to construct a valid TxnID
    Given variable "txnID" is computed from the identity "${identity}" and nonce "${txIDNonce}"
    Given variable "endorseAndCommitRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["put","keyX","valueX"],"tx_id":"${txnID}","nonce":"${txIDNonce}"}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorseandcommit,${endorseAndCommitRequest}" on peers "peer1.org1.example.com"
    Then the JSON path "Status" of the numeric response equals "200"
    Then the JSON path "Committed" of the boolean response equals "true"

    And we wait 5 seconds

    Given variable "endorseRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["get","keyX"]}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorse,${endorseRequest}" on peers "peer1.org2.example.com"
    Then the JSON path "Payload" of the response equals "valueX"

    # CommitEndorsements
    Given variable "endorseRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["put","key1002","value1002"]}'
    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorse,${endorseRequest}" on peers "peer1.org2.example.com"
    Then the JSON path "Committed" of the boolean response equals "false"
    And the JSON path "EndorsementResponse" of the response is saved to variable "endorsementResponse"

    Given variable "commitRequest" is assigned the JSON value '{"endorsement_response":"${endorsementResponse}"}'
    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "commit,${commitRequest}" on peers "peer1.org2.example.com"
    Then the JSON path "Committed" of the boolean response equals "true"

    # Async commit
    Given variable "endorseAndCommitRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["put","key1100","value1100"],"async_commit":true}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorseandcommit,${endorseAndCommitRequest}" on peers "peer0.org2.example.com"
    Then the JSON path "Committed" of the boolean response equals "true"

    And we wait 5 seconds

    Given variable "endorseRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["get","key1002"]}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorse,${endorseRequest}" on peers "peer1.org2.example.com"
    Then the JSON path "Payload" of the response equals "value1002"

    Given variable "endorseRequest" is assigned the JSON value '{"cc_id":"target_cc","args":["get","key1100"]}'
    And txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "endorse,${endorseRequest}" on peers "peer1.org2.example.com"
    Then the JSON path "Payload" of the response equals "value1100"

    # verifyProposalSignature
    Given a signed proposal is created for chaincode "target_cc" with args "put,key1003,value1003" with org "peerorg1" on channel "mychannel" and is saved to variable "signedProposal"
    Then txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "verifyProposalSignature,${signedProposal}" on peers "peer1.org2.example.com"

    Given a signed proposal with an invalid signature is created for chaincode "target_cc" with args "put,key1004,value1004" with org "peerorg1" on channel "mychannel" and is saved to variable "signedProposal"
    Then txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "verifyProposalSignature,${signedProposal}" on peers "peer1.org2.example.com" then the error response should contain "The creator's signature over the proposal is not valid"

    # validateProposalResponses
    Given a signed proposal is created for chaincode "target_cc" with args "put,key1005,value1005" with org "peerorg1" on channel "mychannel" and is saved to variable "signedProposal"
    Then the signed proposal "${signedProposal}" is sent to peers "peer0.org1.example.com,peer0.org2.example.com" and the responses are saved to variable "proposalResponses"
    Then txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "validateProposalResponses,${signedProposal},${proposalResponses}" on peers "peer1.org2.example.com"

    Given a signed proposal is created for chaincode "target_cc" with args "put,key1006,value1006" with org "peerorg1" on channel "mychannel" and is saved to variable "signedProposal"
    Then the signed proposal "${signedProposal}" is sent to peers "peer0.org1.example.com,peer0.org2.example.com" and the invalid responses are saved to variable "proposalResponses"
    Then txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "validateProposalResponses,${signedProposal},${proposalResponses}" on peers "peer1.org2.example.com" then the error response should contain "the creator's signature over the proposal response is not valid"

  @txn_s2
  Scenario: Configuration validation errors
    Given variable "org1ConfigUpdateInvalidTxn" is assigned config from file "./fixtures/config/fabric/org1-config-invalid-txn.json"
    When client queries chaincode "configscc" with args "save,${org1ConfigUpdateInvalidTxn}" on the "mychannel" channel then the error response should contain "validation error: field 'UserID' was not specified for TXN config"

    Given variable "org1ConfigUpdateInvalidSDK" is assigned config from file "./fixtures/config/fabric/org1-config-invalid-sdk.json"
    When client queries chaincode "configscc" with args "save,${org1ConfigUpdateInvalidSDK}" on the "mychannel" channel then the error response should contain "validation error: invalid SDK config"
