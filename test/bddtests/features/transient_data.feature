#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

@all
@transient_data
Feature:

  @transient_data_s1
  Scenario: Put and get transient data
    Given the channel "mychannel" is created and all peers have joined
    And transient collection config "tdata_coll1" is defined for collection "collection1" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and timeToLive=5s
    And transient collection config "tdata_coll2" is defined for collection "collection2" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and timeToLive=10m
    And collection config "coll3" is defined for collection "collection3" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and blocksToLive=1000
    And "test" chaincode "tdata_examplecc" is installed from path "github.com/trustbloc/e2e_cc" to all peers
    And "test" chaincode "tdata_examplecc" is instantiated from path "github.com/trustbloc/e2e_cc" on the "mychannel" channel with args "" with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" with collection policy "tdata_coll1,tdata_coll2,coll3"
    And chaincode "tdata_examplecc" is warmed up on all peers on the "mychannel" channel

    When client queries chaincode "tdata_examplecc" with args "putprivate,collection1,key1,value1" on the "mychannel" channel
    And client queries chaincode "tdata_examplecc" with args "getprivate,collection1,key1" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "value1"

    When client queries chaincode "tdata_examplecc" with args "putprivate,collection2,key2,value2" on the "mychannel" channel
    And client queries chaincode "tdata_examplecc" with args "getprivate,collection2,key2" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "value2"

    # Should not be able to update a value for a key
    When client queries chaincode "tdata_examplecc" with args "putprivate,collection2,key2,value3" on the "mychannel" channel
    And client queries chaincode "tdata_examplecc" with args "getprivate,collection2,key2" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "value2"

    # Test for put of keys on multiple collections. The first two collections are Transient Data type so they should persist.
    # The third is a private data collection, so the data should not persist.
    When client queries chaincode "tdata_examplecc" with args "putprivatemultiple,collection1,keyX,valX,collection2,keyY,valY,collection3,keyZ,valZ" on the "mychannel" channel
    And client queries chaincode "tdata_examplecc" with args "getprivatemultiple,collection1,keyX,collection2,keyY,collection3,keyZ" on the "mychannel" channel
    Then response from "dcas_examplecc" to client equal value "valX,valY,"

    # Test for the case where get transient data should not return data that was stored in the current transaction.
    # When multiple peers are involved in an endorsement then one endorser may already have propagated the private
    # data to another endorser. The first endorser will think that there is no value for a given key but the second
    # endorser will find the value that was provided by the first. In this case, the current transaction ID is compared
    # with the transaction ID at which the transient data was stored. If the IDs match then nil is returned.
    When client queries chaincode "tdata_examplecc" with args "getandputprivate,collection2,key4,value4" on the "mychannel" channel
    And client queries chaincode "tdata_examplecc" with args "getprivate,collection2,key4" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "value4"

    # Since collection3 is a non-transient collection, it requires a commit to be persisted, so we should expect an empty value on query
    When client queries chaincode "tdata_examplecc" with args "putprivate,collection3,key3,value3" on the "mychannel" channel
    And client queries chaincode "tdata_examplecc" with args "getprivate,collection3,key3" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value ""

    # Test expiry
    Given we wait 5 seconds

    # Should have expired
    When client queries chaincode "tdata_examplecc" with args "getprivate,collection1,key1" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value ""

    # Should still be there
    When client queries chaincode "tdata_examplecc" with args "getprivate,collection2,key2" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "value2"

    # Test to make sure private data collections still persist in a transaction and that transient-data reads/writes work with transactions
    When client invokes chaincode "tdata_examplecc" with args "putprivatemultiple,collection1,pvtKey1,pvtVal1,collection2,pvtKey2,pvtVal2,collection3,pvtKey3,pvtVal3" on the "mychannel" channel
    And we wait 2 seconds
    And client queries chaincode "tdata_examplecc" with args "getprivatemultiple,collection1,pvtKey1,collection2,pvtKey2,collection3,pvtKey3" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "pvtVal1,pvtVal2,pvtVal3"

    # Test Chaincode Upgrade and Cache Expiration
    #   When the chaincode is upgraded with a new policy, all caches should be refreshed
    #   Change the policy of collection1 so that it expires in 1m instead of 3s and
    #   change the policy of collection2 so that it expires in 3s instead of 10m
    Given transient collection config "tdata_coll1_upgrade" is defined for collection "collection1" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and timeToLive=1m
    And transient collection config "tdata_coll2_upgrade" is defined for collection "collection2" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and timeToLive=3s
    And collection config "coll3" is defined for collection "collection3" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and blocksToLive=1000
    And "test" chaincode "tdata_examplecc" version "v2" is installed from path "github.com/trustbloc/e2e_cc" to all peers
    And "test" chaincode "tdata_examplecc" is upgraded with version "v2" from path "github.com/trustbloc/e2e_cc" on the "mychannel" channel with args "" with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" with collection policy "tdata_coll1_upgrade,tdata_coll2_upgrade,coll3"
    And chaincode "tdata_examplecc" is warmed up on all peers on the "mychannel" channel

    When client queries chaincode "tdata_examplecc" with args "putprivate,collection1,keyA,valueA" on the "mychannel" channel
    And client queries chaincode "tdata_examplecc" with args "getprivate,collection1,keyA" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "valueA"

    When client queries chaincode "tdata_examplecc" with args "putprivate,collection2,keyB,valueB" on the "mychannel" channel
    And client queries chaincode "tdata_examplecc" with args "getprivate,collection2,keyB" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "valueB"

    # Test expiry
    Given we wait 5 seconds

    # Should still be there
    When client queries chaincode "tdata_examplecc" with args "getprivate,collection1,keyA" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "valueA"

    # Should have expired
    When client queries chaincode "tdata_examplecc" with args "getprivate,collection2,keyB" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value ""
