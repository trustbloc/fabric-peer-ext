#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

@lc_all
@lc_off_ledger
Feature: Lifecycle off-ledger

  @lc_off_ledger_put_and_get
  Scenario: Put and get off-ledger data
    Given the channel "mychannel" is created and all peers have joined
    And the channel "yourchannel" is created and all peers have joined

    # Data in collection1 are purged after 10s
    And off-ledger collection config "ol_coll1" is defined for collection "collection1" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and timeToLive=10s
    # Data in collection2 are never purged
    And DCAS collection config "dcas_coll2" is defined for collection "collection2" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and timeToLive=
    # Data in collection3 are purged after 10 blocks of being added
    And collection config "coll3" is defined for collection "collection3" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=3, and blocksToLive=10
    # Data in accounts are never purged
    And off-ledger collection config "accounts" is defined for collection "accounts" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and timeToLive=
    # Define an off-ledger collection with implicit policy, which means that data is stored and distributed only within the peer's local org.
    And off-ledger collection config "ol_implicitColl" is defined for collection "implicitcoll" as policy="OR('IMPLICIT-ORG.member')", requiredPeerCount=0, maxPeerCount=0, and timeToLive=10s

    Then chaincode "ol_examplecc", version "v1", package ID "ol_examplecc:v1", sequence 1 is approved and committed by orgs "peerorg1,peerorg2" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy "ol_coll1,dcas_coll2,coll3,accounts,ol_implicitColl"
    And chaincode "ol_examplecc_2", version "v1", package ID "ol_examplecc_2:v1", sequence 1 is approved and committed by orgs "peerorg1,peerorg2" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy "ol_coll1,accounts,ol_implicitColl"
    And chaincode "ol_examplecc_2", version "v1", package ID "ol_examplecc_2:v1", sequence 1 is approved and committed by orgs "peerorg1,peerorg2" on the "yourchannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy "ol_coll1,accounts,ol_implicitColl"

    Then we wait 10 seconds

    # Test for invalid CAS key
    When client queries chaincode "ol_examplecc" with args "putprivate,collection2,key1,value1" on the "mychannel" channel then the error response should contain "invalid CAS key [key1]"

    # Set off-ledger data on a peer in one org and get it from one in another org
    When client queries chaincode "ol_examplecc" with args "putprivate,collection1,key1,value1" on a single peer in the "peerorg1" org on the "mychannel" channel
    And client queries chaincode "ol_examplecc" with args "getprivate,collection1,key1" on a single peer in the "peerorg2" org on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "value1"

    # Set DCAS data on a peer in one org and get it from one in another org
    When client queries chaincode "ol_examplecc" with args "putcas,collection2,value2" on a single peer in the "peerorg2" org on the "mychannel" channel
    And the response is saved to variable "key2"
    And client queries chaincode "ol_examplecc" with args "getcas,collection2,${key2}" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "value2"
    And client queries chaincode "ol_examplecc" with args "getcas,collection2,${key2}" on a single peer in the "peerorg2" org on the "mychannel" channel
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
    And client queries chaincode "ol_examplecc" with args "getcas,collection2,${key4}" on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "value4"

    # Test expiry
    Given we wait 15 seconds

    # Should have expired
    When client queries chaincode "ol_examplecc" with args "getprivate,collection1,key1" on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value ""

    # Should still be there
    When client queries chaincode "ol_examplecc" with args "getcas,collection2,${key2}" on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "value2"

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

    Then chaincode "ol_examplecc", version "v1.0.1", package ID "ol_examplecc:v1", sequence 2 is approved and committed by orgs "peerorg1,peerorg2" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy "ol_coll1_upgrade,dcas_coll2_upgrade,coll3,accounts,ol_implicitColl"
    Then we wait 5 seconds

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
    And client queries chaincode "ol_examplecc" with args "getcas,collection2,${keyB}" on a single peer in the "peerorg2" org on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "valueB"
    # Get the data from org1 - the data should NOT be there
    When client queries chaincode "ol_examplecc" with args "getcas,collection2,${keyB}" on a single peer in the "peerorg1" org on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value ""

    # Test chaincode-to-chaincode invocation (same channel)
    # Set the data on the target chaincode using a chaincode-to-chaincode invocation
    When client queries chaincode "ol_examplecc" with args "invokecc,ol_examplecc_2,,{`Args`:[`putprivate`|`collection1`|`keyC`|`valueC`]}" on the "mychannel" channel
    # Query the target chaincode directly
    And client queries chaincode "ol_examplecc_2" with args "getprivate,collection1,keyC" on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "valueC"
    # Query the target chaincode using a chaincode-to-chaincode invocation (same channel)
    When client queries chaincode "ol_examplecc" with args "invokecc,ol_examplecc_2,,{`Args`:[`getprivate`|`collection1`|`keyC`]}" on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "valueC"

    # Test chaincode-to-chaincode invocation (different channel)
    # Set the data on a different channel
    When client queries chaincode "ol_examplecc_2" with args "putprivate,collection1,keyD,valueD" on the "yourchannel" channel
    # Query the target chaincode directly
    And client queries chaincode "ol_examplecc_2" with args "getprivate,collection1,keyD" on the "yourchannel" channel
    Then response from "ol_examplecc_2" to client equal value "valueD"
    # Query the target chaincode using a chaincode-to-chaincode invocation
    When client queries chaincode "ol_examplecc" with args "invokecc,ol_examplecc_2,yourchannel,{`Args`:[`getprivate`|`collection1`|`keyD`]}" on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "valueD"

    # Rich Queries
    Given the account with ID "456456", owner "Tim Jones" and a balance of 1000 is created and stored to variable "tim_account"
    And client queries chaincode "ol_examplecc" with args "putprivate,accounts,account1,${tim_account}" on the "mychannel" channel
    When client queries chaincode "ol_examplecc" with args "queryprivate,accounts,{`selector`:{`id`:`456456`}|`fields`:[`id`|`balance`|`owner`|`operationType`|`order`]|`use_index`:[`_design/indexIDDoc`|`indexID`]}" on the "mychannel" channel
    And the response is saved to variable "account_operations"
    Then the variable "account_operations" contains 1 accounts
    And the variable "account_operations" contains an account at index 0 with Key "account1", ID "456456", Owner "Tim Jones", and Balance 1000

    Given the account stored in variable "tim_account" is updated with a balance of 2000
    And client queries chaincode "ol_examplecc" with args "putprivate,accounts,account2,${tim_account}" on the "mychannel" channel
    When client queries chaincode "ol_examplecc" with args "queryprivate,accounts,{`selector`:{`id`:`456456`}|`fields`:[`id`|`balance`|`owner`|`operationType`|`order`]|`use_index`:[`_design/indexIDDoc`|`indexID`]}" on the "mychannel" channel
    And the response is saved to variable "account_operations"
    Then the variable "account_operations" contains 2 accounts
    And the variable "account_operations" contains an account at index 0 with Key "account1", ID "456456", Owner "Tim Jones", and Balance 1000
    And the variable "account_operations" contains an account at index 1 with Key "account2", ID "456456", Owner "Tim Jones", and Balance 2000

    # Test for off-ledger collections with implicit policy, i.e. data is stored and distributed only within the peer's local org.
    When client queries chaincode "ol_examplecc" with args "putprivate,implicitcoll,key1,org1value" on peers "peer1.org1.example.com" on the "mychannel" channel
    And client queries chaincode "ol_examplecc" with args "putprivate,implicitcoll,key1,org2value" on peers "peer1.org2.example.com" on the "mychannel" channel

    When client queries chaincode "ol_examplecc" with args "getprivate,implicitcoll,key1" on peers "peer0.org1.example.com,peer1.org1.example.com" on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "org1value"

    When client queries chaincode "ol_examplecc" with args "getprivate,implicitcoll,key1" on peers "peer0.org2.example.com,peer1.org2.example.com" on the "mychannel" channel
    Then response from "ol_examplecc" to client equal value "org2value"

  @lc_dcas_client
  Scenario: DCAS Client
    Given the channel "mychannel" is created and all peers have joined

    And DCAS collection config "dcas_coll" is defined for collection "collection1" as policy="OR('Org1MSP.member','Org2MSP.member')", requiredPeerCount=1, maxPeerCount=2, and timeToLive=
    Then chaincode "dcas_examplecc", version "v1", package ID "ol_examplecc:v1", sequence 1 is approved and committed by orgs "peerorg1,peerorg2" on the "mychannel" channel with endorsement policy "AND('Org1MSP.member','Org2MSP.member')" and collection policy "dcas_coll"

    And we wait 5 seconds

    # ------ Store data as a file
    # Store a small file that fits in one node
    Given variable "fileData1" is assigned the value "File data #1"
    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "putcas,dcas_examplecc,collection1,${fileData1},node-type=file" on peers "peer0.org2.example.com"
    Then the response is saved to variable "file1_cid"
    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "getcas,dcas_examplecc,collection1,${file1_cid}" on peers "peer0.org1.example.com"
    Then response from "client" to client equal value "${fileData1}"
    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "getcasnode,dcas_examplecc,collection1,${file1_cid}" on peers "peer0.org1.example.com"
    # The contents of a small file should be stored in a single raw node
    Then the JSON path "data" of the response is not empty
    And the JSON path "links" of the response has 0 items

    # Store a large file that's broken up into raw nodes of 32 bytes each and the root node links to these raw nodes
    # (Note that the block size is set to 32 bytes for this test in docker-compose-base.yml: CORE_COLL_DCAS_MAXBLOCKSIZE=32)
    Given variable "fileData2" is assigned the value "Here is some data which is longer than the maximum size for a file node in DCAS so it will have to be split into multiple chunks."
    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "putcas,dcas_examplecc,collection1,${fileData2},node-type=file" on peers "peer0.org2.example.com"
    Then the response is saved to variable "file2_cid"
    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "getcas,dcas_examplecc,collection1,${file2_cid}" on peers "peer0.org1.example.com"
    Then response from "client" to client equal value "${fileData2}"
    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "getcasnode,dcas_examplecc,collection1,${file2_cid}" on peers "peer0.org1.example.com"
    Then the JSON path "data" of the response is not empty
    And the JSON path "links" of the array response is not empty
    And the JSON path "links.0.size" of the numeric response equals "32"

    # ------ Store data as an object
    # This JSON object contains arbitrary fields as well as links to the files that were created above
    Given variable "jsonData" is assigned the uncanonicalized JSON value '{"field1":"value1","files":[{"/":"${file1_cid}"},{"/":"${file2_cid}"}]}'
    # This JSON object is the same as the one above but with the fields in different order
    And variable "jsonData2" is assigned the uncanonicalized JSON value '{"files":[{"/":"${file1_cid}"},{"/":"${file2_cid}"}],"field1":"value1"}'

    # Store object in CBOR format
    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "putcas,dcas_examplecc,collection1,${jsonData},node-type=object;input-encoding=json;format=cbor" on peers "peer0.org2.example.com"
    Then the response is saved to variable "cid1"
    # Store the same JSON object which has the fields out of order. This JSON object should be treated exactly the same as the object above when stored in CBOR format since JSON objects are canonicalized before they are stored.
    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "putcas,dcas_examplecc,collection1,${jsonData2},node-type=object;input-encoding=json;format=cbor" on peers "peer0.org2.example.com"
    Then response from "" to client equal value "${cid1}"

    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "getcas,dcas_examplecc,collection1,${cid1}" on peers "peer0.org1.example.com"
    Then the JSON path "field1" of the response equals "value1"
    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "getcasnode,dcas_examplecc,collection1,${cid1}" on peers "peer0.org1.example.com"
    Then the JSON path "data" of the response is not empty
    Then the JSON path "links.0.hash" of the response equals "${file1_cid}"
    Then the JSON path "links.1.hash" of the response equals "${file2_cid}"

    # Store object in raw format
    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "putcas,dcas_examplecc,collection1,${jsonData},node-type=object;input-encoding=raw;format=raw" on peers "peer0.org2.example.com"
    Then the response is saved to variable "cid2"
    # Store the same JSON object which has the fields out of order. This JSON object should NOT be treated the same as the object above when stored in RAW format since the object is not canonicalized before being stored.
    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "putcas,dcas_examplecc,collection1,${jsonData2},node-type=object;input-encoding=raw;format=raw" on peers "peer0.org2.example.com"
    Then the response is saved to variable "cid2_1"
    Then the value "${cid2_1}" does not equal "${cid2}"

    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "getcas,dcas_examplecc,collection1,${cid2}" on peers "peer0.org1.example.com"
    Then the JSON path "field1" of the response equals "value1"
    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "getcasnode,dcas_examplecc,collection1,${cid2}" on peers "peer0.org1.example.com"
    Then the JSON path "data" of the response is not empty
    # Since the data was saved in raw format, the links are not interpreted
    And the JSON path "links" of the response has 0 items

    # Store object in protobuf format
    Given variable "pb-data" is assigned the JSON value '{"data":"","links":[{"Cid":{"/":"${file1_cid}"}},{"Cid":{"/":"${file2_cid}"}}]}'
    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "putcas,dcas_examplecc,collection1,${pb-data},node-type=object;input-encoding=json;format=protobuf" on peers "peer0.org2.example.com"
    Then the response is saved to variable "cid3"
    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "getcasnode,dcas_examplecc,collection1,${cid3}" on peers "peer0.org1.example.com"
    Then the JSON path "data" of the response is not empty
    Then the JSON path "links.0.hash" of the response equals "${file1_cid}"
    Then the JSON path "links.1.hash" of the response equals "${file2_cid}"

    # Unsupported CID encoding error
    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "getcas,dcas_examplecc,collection1,invalid-cid" on peers "peer0.org1.example.com" then the error response should contain "selected encoding not supported"

    # CIDV0 (base58) encoding
    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "getcas,dcas_examplecc,collection1,QmSvs5BUivE2yhuWN8dmn96eCPr1d6S7ZMTpPYaQJPDJ16" on peers "peer0.org1.example.com"
    Then response from "client" to client equal value ""

    # CIDV1 (base32) encoding
    When txn service is invoked on channel "mychannel" with chaincode "e2e_cc" with args "getcas,dcas_examplecc,collection1,bafybeib3ykh7ipd3fz53bt44laq2xjqm5zly5ya33wx27vqnq737jimk2i" on peers "peer0.org1.example.com"
    Then response from "client" to client equal value ""
