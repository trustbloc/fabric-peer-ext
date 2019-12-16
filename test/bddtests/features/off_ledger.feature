#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

@all
@off_ledger
Feature: off-ledger

  Background: Setup
    Given the channel "mychannel" is created and all peers have joined
    And the channel "yourchannel" is created and all peers have joined

    And off-ledger collection config "ol_coll1" is defined for collection "collection1" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=3, and timeToLive=10s
    And DCAS collection config "dcas_coll2" is defined for collection "collection2" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=3, and timeToLive=10m
    And collection config "coll3" is defined for collection "collection3" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=3, and blocksToLive=10
    And DCAS collection config "dcas_accounts" is defined for collection "accounts" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=3, and timeToLive=30m
    And "test" chaincode "ol_examplecc" is instantiated from path "in-process" on the "mychannel" channel with args "" with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" with collection policy "ol_coll1,dcas_coll2,coll3,dcas_accounts"

  @off_ledger_s1
  Scenario: Put and get off-ledger data
    # Test for invalid CAS key
    When client queries chaincode "ol_examplecc" with args "putprivate,collection2,key1,value1" on the "mychannel" channel then the error response should contain "the key should be the hash of the value"

    # Set off-ledger data on a peer in one org and get it from one in another org
    When client queries chaincode "ol_examplecc" with args "putprivate,collection1,key1,value1" on a single peer in the "peerorg1" org on the "mychannel" channel
    And client queries chaincode "ol_examplecc" with args "getprivate,collection1,key1" on a single peer in the "peerorg2" org on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "value1"

    # Set DCAS data on a peer in one org and get it from one in another org
    When client queries chaincode "ol_examplecc" with args "putcas,collection2,value2" on a single peer in the "peerorg2" org on the "mychannel" channel
    And the response is saved to variable "key2"
    And client queries chaincode "ol_examplecc" with args "getprivate,collection2,${key2}" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "value2"
    And client queries chaincode "ol_examplecc" with args "getprivate,collection2,${key2}" on a single peer in the "peerorg2" org on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "value2"

    # Test for put of keys on multiple collections. The first two collections are off-ledger/DCAS type so they should persist. The third
    # is a private data collection, so the data should not persist.
    Given variable "casKeyY" is assigned the CAS key of value "valY"
    When client queries chaincode "ol_examplecc" with args "putprivatemultiple,collection1,keyX,valX,collection2,${casKeyY},valY,collection3,keyZ,valZ" on the "mychannel" channel
    And client queries chaincode "ol_examplecc" with args "getprivatemultiple,collection1,keyX,collection2,${casKeyY},collection3,keyZ" on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "valX,valY,"

    # Test for the case where get data should not return data that was stored in the current transaction.
    # When multiple peers are involved in an endorsement then one endorser may already have propagated the private
    # data to another endorser. The first endorser will think that there is no value for a given key but the second
    # endorser will find the value that was provided by the first. In this case, the current transaction ID is compared
    # with the transaction ID at which the data was stored. If the IDs match then nil is returned.
    When client queries chaincode "ol_examplecc" with args "getandputcas,collection2,value4" on the "mychannel" channel
    And the response is saved to variable "key4"
    And client queries chaincode "ol_examplecc" with args "getprivate,collection2,${key4}" on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "value4"

    # Test expiry
    Given we wait 15 seconds

    # Should have expired
    When client queries chaincode "ol_examplecc" with args "getprivate,collection1,key1" on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value ""

    # Should still be there
    When client queries chaincode "ol_examplecc" with args "getprivate,collection2,${key2}" on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "value2"

    # Delete the data on one peer - should be deleted from both peers (TODO: need to be tested with cache enabled)
    When client queries chaincode "ol_examplecc" with args "delprivate,collection2,${key2}" on a single peer in the "peerorg1" org on the "mychannel" channel
    And client queries chaincode "ol_examplecc" with args "getprivate,collection2,${key2}" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value ""
    And client queries chaincode "ol_examplecc" with args "getprivate,collection2,${key2}" on a single peer in the "peerorg2" org on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value ""

    # Test to make sure private data collections still persist in a transaction and that off-ledger reads/writes work with transactions
    Given variable "pvtKey2" is assigned the CAS key of value "pvtVal2"
    When client invokes chaincode "ol_examplecc" with args "putprivatemultiple,collection1,pvtKey1,pvtVal1,collection2,${pvtKey2},pvtVal2,collection3,pvtKey3,pvtVal3" on the "mychannel" channel
    And we wait 2 seconds
    And client queries chaincode "ol_examplecc" with args "getprivatemultiple,collection1,pvtKey1,collection2,${pvtKey2},collection3,pvtKey3" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "pvtVal1,pvtVal2,pvtVal3"

    # Test Chaincode Upgrade
    #   Change the policy of the DCAS collections to disseminate to only one org. When the chaincode is upgraded
    #   then all caches should be refreshed.
    Given off-ledger collection config "ol_coll1_upgrade" is defined for collection "collection1" as policy="OR('Org1MSP.member')", requiredPeerCount=1, maxPeerCount=2, and timeToLive=1m
    And DCAS collection config "dcas_coll2_upgrade" is defined for collection "collection2" as policy="OR('Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and timeToLive=1m
    And "test" chaincode "ol_examplecc" is upgraded with version "v3" from path "in-process" on the "mychannel" channel with args "" with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" with collection policy "ol_coll1_upgrade,dcas_coll2_upgrade,coll3,dcas_accounts"
    # Put the data to org2 - the data should be disseminated to org1
    When client queries chaincode "ol_examplecc" with args "putprivate,collection1,keyA,valueA" on a single peer in the "peerorg2" org on the "mychannel" channel
    And we wait 1 seconds
    # Get the data from org1 - the data should be there
    And client queries chaincode "ol_examplecc" with args "getprivate,collection1,keyA" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "valueA"
    # Get the data from org2 - the data should NOT be there
    When client queries chaincode "ol_examplecc" with args "getprivate,collection1,keyA" on a single peer in the "peerorg2" org on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value ""
    # Put the data to org1 - the data should be disseminated to org2
    When client queries chaincode "ol_examplecc" with args "putcas,collection2,valueB" on a single peer in the "peerorg1" org on the "mychannel" channel
    And the response is saved to variable "keyB"
    And we wait 1 seconds
    # Get the data from org2 - the data should be there
    And client queries chaincode "ol_examplecc" with args "getprivate,collection2,${keyB}" on a single peer in the "peerorg2" org on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "valueB"
    # Get the data from org1 - the data should NOT be there
    When client queries chaincode "ol_examplecc" with args "getprivate,collection2,${keyB}" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value ""

    # Test chaincode-to-chaincode invocation (same channel)
    Given "test" chaincode "ol_examplecc_2" is instantiated from path "in-process" on the "mychannel" channel with args "" with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" with collection policy "ol_coll1,dcas_accounts"
    # Set the data on the target chaincode using a chaincode-to-chaincode invocation
    When client queries chaincode "ol_examplecc" with args "invokecc,ol_examplecc_2,,{`Args`:[`putprivate`|`collection1`|`keyC`|`valueC`]}" on the "mychannel" channel
    # Query the target chaincode directly
    And client queries chaincode "ol_examplecc_2" with args "getprivate,collection1,keyC" on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "valueC"
    # Query the target chaincode using a chaincode-to-chaincode invocation (same channel)
    When client queries chaincode "ol_examplecc" with args "invokecc,ol_examplecc_2,,{`Args`:[`getprivate`|`collection1`|`keyC`]}" on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "valueC"

    # Test chaincode-to-chaincode invocation (different channel)
    Given "test" chaincode "ol_examplecc_2" is instantiated from path "in-process" on the "yourchannel" channel with args "" with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" with collection policy "ol_coll1,dcas_accounts"
    # Set the data on a different channel
    When client queries chaincode "ol_examplecc_2" with args "putprivate,collection1,keyD,valueD" on the "yourchannel" channel
    # Query the target chaincode directly
    And client queries chaincode "ol_examplecc_2" with args "getprivate,collection1,keyD" on the "yourchannel" channel
    Then response from "ol_examplecc_2" to client equal value "valueD"
    # Query the target chaincode using a chaincode-to-chaincode invocation
    When client queries chaincode "ol_examplecc" with args "invokecc,ol_examplecc_2,yourchannel,{`Args`:[`getprivate`|`collection1`|`keyD`]}" on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "valueD"

  @off_ledger_s2
  Scenario: Rich Queries
    Given the account with ID "456456", owner "Tim Jones" and a balance of 1000 is created and stored to variable "tim_account"
    And client queries chaincode "ol_examplecc" with args "putcas,accounts,${tim_account}" on the "mychannel" channel
    And the response is saved to variable "tim_account_key"
    When client queries chaincode "ol_examplecc" with args "queryprivate,accounts,{`selector`:{`id`:`456456`}|`fields`:[`id`|`balance`|`owner`|`operationType`|`order`]|`use_index`:[`_design/indexIDDoc`|`indexID`]}" on the "mychannel" channel
    And the response is saved to variable "account_operations"
    Then the variable "account_operations" contains 1 accounts
    And the variable "account_operations" contains an account at index 0 with Key "${tim_account_key}", ID "456456", Owner "Tim Jones", and Balance 1000

    Given the account stored in variable "tim_account" is updated with a balance of 2000
    And client queries chaincode "ol_examplecc" with args "putcas,accounts,${tim_account}" on the "mychannel" channel
    And the response is saved to variable "tim_account_update_key"
    When client queries chaincode "ol_examplecc" with args "queryprivate,accounts,{`selector`:{`id`:`456456`}|`fields`:[`id`|`balance`|`owner`|`operationType`|`order`]|`use_index`:[`_design/indexIDDoc`|`indexID`]}" on the "mychannel" channel
    And the response is saved to variable "account_operations"
    Then the variable "account_operations" contains 2 accounts
    And the variable "account_operations" contains an account at index 0 with Key "${tim_account_key}", ID "456456", Owner "Tim Jones", and Balance 1000
    And the variable "account_operations" contains an account at index 1 with Key "${tim_account_update_key}", ID "456456", Owner "Tim Jones", and Balance 2000
