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
    And the channel "yourchannel" is created and all peers have joined

    And transient collection config "tdata_coll1" is defined for collection "collection1" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and timeToLive=5s
    And transient collection config "tdata_coll2" is defined for collection "collection2" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and timeToLive=10m
    And collection config "coll3" is defined for collection "collection3" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and blocksToLive=1000

    # Start out with a chaincode that just has one static collection
    And "test" chaincode "tdata_examplecc" is instantiated from path "in-process" on the "mychannel" channel with args "" with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" with collection policy "coll3"

    # Prove that the transient data collection1 is not there
    When client queries chaincode "tdata_examplecc" with args "putprivate,collection1,key1,value1" on the "mychannel" channel then the error response should contain "collection mychannel/tdata_examplecc/collection1 could not be found"
    When client invokes chaincode "tdata_examplecc" with args "putprivate,collection3,pvtKeyX,pvtValX" on the "mychannel" channel
    And we wait 2 seconds
    And client queries chaincode "tdata_examplecc" with args "getprivate,collection3,pvtKeyX" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "pvtValX"

    # Upgrade the chaincode, adding two transient data collections
    Given "test" chaincode "tdata_examplecc" is upgraded with version "v2" from path "in-process" on the "mychannel" channel with args "" with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" with collection policy "tdata_coll1,tdata_coll2,coll3"
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
    Then response from "tdata_examplecc" to client equal value "valX,valY,"

    # Test for put of keys on multiple transient data collections within the same endorsement.
    When client queries chaincode "tdata_examplecc" with args "putprivatemultiple,collection2,key_0,val_0,collection2,key_1,val_1,collection2,key_2,val_2,collection2,key_3,val_3,collection2,key_4,val_4,collection2,key_5,val_5,collection2,key_6,val_6,collection2,key_7,val_7,collection2,key_8,val_8,collection2,key_9,val_9" on the "mychannel" channel
    And client queries chaincode "tdata_examplecc" with args "getprivate,collection2,key_0" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "val_0"
    And client queries chaincode "tdata_examplecc" with args "getprivate,collection2,key_1" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "val_1"
    And client queries chaincode "tdata_examplecc" with args "getprivate,collection2,key_2" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "val_2"
    And client queries chaincode "tdata_examplecc" with args "getprivate,collection2,key_3" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "val_3"
    And client queries chaincode "tdata_examplecc" with args "getprivate,collection2,key_4" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "val_4"
    And client queries chaincode "tdata_examplecc" with args "getprivate,collection2,key_5" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "val_5"
    And client queries chaincode "tdata_examplecc" with args "getprivate,collection2,key_6" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "val_6"
    And client queries chaincode "tdata_examplecc" with args "getprivate,collection2,key_7" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "val_7"
    And client queries chaincode "tdata_examplecc" with args "getprivate,collection2,key_8" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "val_8"
    And client queries chaincode "tdata_examplecc" with args "getprivate,collection2,key_9" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "val_9"

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
    Then response from "tdata_examplecc" to client equal value "pvtVal1,pvtVal2,pvtVal3"
    # Ensure that a write to one collection and read from another works (issue #158)
    When client invokes chaincode "tdata_examplecc" with args "putprivate,collection3,keyC,valueA" on the "mychannel" channel
    And we wait 5 seconds
    When client invokes chaincode "tdata_examplecc" with args "putprivatemultiple,collection1,key1,value1,collection3,keyC," on the "mychannel" channel

    # Test Chaincode Upgrade:
    #   - Ensure collection Type cannot be changed during chaincode upgrade
    Given transient collection config "coll3_upgrade" is defined for collection "collection3" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and timeToLive=1m
    And "test" chaincode "tdata_examplecc" is upgraded with version "v999" from path "in-process" on the "mychannel" channel with args "" with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" with collection policy "tdata_coll1,tdata_coll2,coll3_upgrade" then the error response should contain "ENDORSEMENT_POLICY_FAILURE. Description: instantiateOrUpgradeCC failed"
    #   - When the chaincode is upgraded with a new policy, all caches should be refreshed.
    #     Change the policy of collection1 so that it expires in 1m instead of 3s and
    #     change the policy of collection2 so that it expires in 3s instead of 10m
    Given transient collection config "tdata_coll1_upgrade" is defined for collection "collection1" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and timeToLive=1m
    And transient collection config "tdata_coll2_upgrade" is defined for collection "collection2" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and timeToLive=3s
    And collection config "coll3" is defined for collection "collection3" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and blocksToLive=1000
    And "test" chaincode "tdata_examplecc" is upgraded with version "v3" from path "in-process" on the "mychannel" channel with args "" with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" with collection policy "tdata_coll1_upgrade,tdata_coll2_upgrade,coll3"

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

    # Test chaincode-to-chaincode invocation (same channel)
    Given "test" chaincode "tdata_examplecc_2" is instantiated from path "in-process" on the "mychannel" channel with args "" with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" with collection policy "tdata_coll2"
    # Set the transient data on the target chaincode using a chaincode-to-chaincode invocation
    When client queries chaincode "tdata_examplecc" with args "invokecc,tdata_examplecc_2,,{`Args`:[`putprivate`|`collection2`|`keyC`|`valueC`]}" on the "mychannel" channel
    # Query the target chaincode directly
    And client queries chaincode "tdata_examplecc_2" with args "getprivate,collection2,keyC" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "valueC"
    # Query the target chaincode using a chaincode-to-chaincode invocation (same channel)
    When client queries chaincode "tdata_examplecc" with args "invokecc,tdata_examplecc_2,,{`Args`:[`getprivate`|`collection2`|`keyC`]}" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "valueC"

    # Test chaincode-to-chaincode invocation (different channel)
    Given "test" chaincode "tdata_examplecc_2" is instantiated from path "in-process" on the "yourchannel" channel with args "" with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" with collection policy "tdata_coll2"
    # Set the transient data on a different channel
    When client queries chaincode "tdata_examplecc_2" with args "putprivate,collection2,keyD,valueD" on the "yourchannel" channel
    # Query the target chaincode directly
    And client queries chaincode "tdata_examplecc_2" with args "getprivate,collection2,keyD" on the "yourchannel" channel
    Then response from "tdata_examplecc_2" to client equal value "valueD"
    # Query the target chaincode using a chaincode-to-chaincode invocation
    When client queries chaincode "tdata_examplecc" with args "invokecc,tdata_examplecc_2,yourchannel,{`Args`:[`getprivate`|`collection2`|`keyD`]}" on the "mychannel" channel
    Then response from "tdata_examplecc" to client equal value "valueD"
